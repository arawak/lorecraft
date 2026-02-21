package validate

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"lorecraft/internal/config"
	"lorecraft/internal/store"
)

type mockStore struct {
	entities      []store.EntitySummary
	entityDetails map[string]*store.Entity
	placeholders  []store.EntitySummary
	orphans       []store.EntitySummary
	duplicates    []store.EntitySummary
	crossLayer    []store.EntitySummary
}

func (m *mockStore) Close(ctx context.Context) error { return nil }

func (m *mockStore) EnsureSchema(ctx context.Context, schema *config.Schema) error { return nil }

func (m *mockStore) UpsertEntity(ctx context.Context, e store.EntityInput) error { return nil }

func (m *mockStore) UpsertRelationship(ctx context.Context, fromName, fromLayer, toName, toLayer, relType string) error {
	return nil
}

func (m *mockStore) RemoveStaleNodes(ctx context.Context, layer string, currentSourceFiles []string) (int64, error) {
	return 0, nil
}

func (m *mockStore) GetLayerHashes(ctx context.Context, layer string) (map[string]string, error) {
	return nil, nil
}

func (m *mockStore) FindEntityLayer(ctx context.Context, name string, layers []string) (string, error) {
	return "", nil
}

func (m *mockStore) ListEntities(ctx context.Context, entityType, layer, tag string) ([]store.EntitySummary, error) {
	return m.entities, nil
}

func (m *mockStore) GetEntity(ctx context.Context, name, entityType string) (*store.Entity, error) {
	if m.entityDetails == nil {
		return nil, nil
	}
	key := name + "|" + entityType
	if entity, ok := m.entityDetails[key]; ok {
		return entity, nil
	}
	return nil, nil
}

func (m *mockStore) GetRelationships(ctx context.Context, name, relType, direction string, depth int) ([]store.Relationship, error) {
	return nil, nil
}

func (m *mockStore) Search(ctx context.Context, query, layer, entityType string) ([]store.SearchResult, error) {
	return nil, nil
}

func (m *mockStore) GetCurrentState(ctx context.Context, name, layer string) (*store.CurrentState, error) {
	return nil, nil
}

func (m *mockStore) GetTimeline(ctx context.Context, layer, entity string, fromSession, toSession int) ([]store.Event, error) {
	return nil, nil
}

func (m *mockStore) ListDanglingPlaceholders(ctx context.Context) ([]store.EntitySummary, error) {
	return m.placeholders, nil
}

func (m *mockStore) ListOrphanedEntities(ctx context.Context) ([]store.EntitySummary, error) {
	return m.orphans, nil
}

func (m *mockStore) ListDuplicateNames(ctx context.Context) ([]store.EntitySummary, error) {
	return m.duplicates, nil
}

func (m *mockStore) ListCrossLayerViolations(ctx context.Context) ([]store.EntitySummary, error) {
	return m.crossLayer, nil
}

func (m *mockStore) RunSQL(ctx context.Context, query string, params map[string]any) ([]map[string]any, error) {
	return nil, nil
}

func TestRun_EnumViolation(t *testing.T) {
	schema := loadSchema(t, `version: 1
entity_types:
  - name: npc
    properties:
      - { name: status, type: enum, values: [alive, dead] }
relationship_types:
  - name: RELATED_TO
`)

	validator := &mockStore{
		entities: []store.EntitySummary{{Name: "Test NPC", EntityType: "npc", Layer: "setting"}},
		entityDetails: map[string]*store.Entity{
			"Test NPC|npc": {
				Name:       "Test NPC",
				EntityType: "npc",
				Layer:      "setting",
				SourceFile: "lore/test.md",
				Properties: map[string]any{"status": "ghost"},
			},
		},
	}

	report, err := Run(context.Background(), schema, validator)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !hasIssueCode(report.Issues, codeEnumInvalid) {
		t.Fatalf("expected enum violation issue")
	}
}

func TestRun_MissingRequiredProperty(t *testing.T) {
	schema := loadSchema(t, `version: 1
entity_types:
  - name: npc
    properties:
      - { name: role, type: string, required: true }
relationship_types:
  - name: RELATED_TO
`)

	validator := &mockStore{
		entities: []store.EntitySummary{{Name: "Test NPC", EntityType: "npc", Layer: "setting"}},
		entityDetails: map[string]*store.Entity{
			"Test NPC|npc": {
				Name:       "Test NPC",
				EntityType: "npc",
				Layer:      "setting",
				SourceFile: "lore/test.md",
				Properties: map[string]any{},
			},
		},
	}

	report, err := Run(context.Background(), schema, validator)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !hasIssueCode(report.Issues, codeMissingRequired) {
		t.Fatalf("expected missing required property issue")
	}
}

func TestRun_DanglingPlaceholder(t *testing.T) {
	schema := loadSchema(t, `version: 1
entity_types:
  - name: npc
relationship_types:
  - name: RELATED_TO
`)

	validator := &mockStore{
		placeholders: []store.EntitySummary{{Name: "Missing NPC", Layer: "setting"}},
	}

	report, err := Run(context.Background(), schema, validator)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !hasIssueCode(report.Issues, codeDanglingPlaceholder) {
		t.Fatalf("expected dangling placeholder issue")
	}
}

func TestRun_OrphanedEntity(t *testing.T) {
	schema := loadSchema(t, `version: 1
entity_types:
  - name: npc
relationship_types:
  - name: RELATED_TO
`)

	validator := &mockStore{
		orphans: []store.EntitySummary{{Name: "Lonely NPC", Layer: "setting"}},
	}

	report, err := Run(context.Background(), schema, validator)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !hasIssueCode(report.Issues, codeOrphanedEntity) {
		t.Fatalf("expected orphaned entity issue")
	}
}

func hasIssueCode(issues []Issue, code string) bool {
	for _, issue := range issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}

func loadSchema(t *testing.T, contents string) *config.Schema {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "schema.yaml")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write schema: %v", err)
	}
	schema, err := config.LoadSchema(path)
	if err != nil {
		t.Fatalf("load schema: %v", err)
	}
	return schema
}
