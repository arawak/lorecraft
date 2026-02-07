package validate

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"lorecraft/internal/config"
	"lorecraft/internal/graph"
)

type mockGraphValidator struct {
	entities      []graph.EntitySummary
	entityDetails map[string]*graph.Entity
	placeholders  []graph.EntitySummary
	orphans       []graph.EntitySummary
	duplicates    []graph.EntitySummary
	crossLayer    []graph.EntitySummary
}

func (m *mockGraphValidator) ListEntities(ctx context.Context, entityType, layer, tag string) ([]graph.EntitySummary, error) {
	return m.entities, nil
}

func (m *mockGraphValidator) GetEntity(ctx context.Context, name, entityType string) (*graph.Entity, error) {
	if m.entityDetails == nil {
		return nil, nil
	}
	key := name + "|" + entityType
	if entity, ok := m.entityDetails[key]; ok {
		return entity, nil
	}
	return nil, nil
}

func (m *mockGraphValidator) ListDanglingPlaceholders(ctx context.Context) ([]graph.EntitySummary, error) {
	return m.placeholders, nil
}

func (m *mockGraphValidator) ListOrphanedEntities(ctx context.Context) ([]graph.EntitySummary, error) {
	return m.orphans, nil
}

func (m *mockGraphValidator) ListDuplicateNames(ctx context.Context) ([]graph.EntitySummary, error) {
	return m.duplicates, nil
}

func (m *mockGraphValidator) ListCrossLayerViolations(ctx context.Context) ([]graph.EntitySummary, error) {
	return m.crossLayer, nil
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

	validator := &mockGraphValidator{
		entities: []graph.EntitySummary{{Name: "Test NPC", EntityType: "npc", Layer: "setting"}},
		entityDetails: map[string]*graph.Entity{
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

	validator := &mockGraphValidator{
		entities: []graph.EntitySummary{{Name: "Test NPC", EntityType: "npc", Layer: "setting"}},
		entityDetails: map[string]*graph.Entity{
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

	validator := &mockGraphValidator{
		placeholders: []graph.EntitySummary{{Name: "Missing NPC", Layer: "setting"}},
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

	validator := &mockGraphValidator{
		orphans: []graph.EntitySummary{{Name: "Lonely NPC", Layer: "setting"}},
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
