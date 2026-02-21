package ingest

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"lorecraft/internal/config"
	"lorecraft/internal/store"
)

type mockStore struct {
	entities      []store.EntityInput
	relationships []struct {
		fromName  string
		fromLayer string
		toName    string
		toLayer   string
		relType   string
	}
	removeCalls []struct {
		layer string
		files []string
	}
	ensureCalled bool
	failUpsert   bool
	layerHashes  map[string]map[string]string
	entityLayers map[string]map[string]struct{}
}

func (m *mockStore) Close(ctx context.Context) error { return nil }

func (m *mockStore) EnsureSchema(ctx context.Context, schema *config.Schema) error {
	m.ensureCalled = true
	return nil
}

func (m *mockStore) UpsertEntity(ctx context.Context, e store.EntityInput) error {
	if m.failUpsert && e.Name == "Test NPC" {
		return errors.New("forced error")
	}
	m.entities = append(m.entities, e)
	return nil
}

func (m *mockStore) UpsertRelationship(ctx context.Context, fromName, fromLayer, toName, toLayer, relType string) error {
	m.relationships = append(m.relationships, struct {
		fromName  string
		fromLayer string
		toName    string
		toLayer   string
		relType   string
	}{fromName: fromName, fromLayer: fromLayer, toName: toName, toLayer: toLayer, relType: relType})
	return nil
}

func (m *mockStore) RemoveStaleNodes(ctx context.Context, layer string, currentSourceFiles []string) (int64, error) {
	m.removeCalls = append(m.removeCalls, struct {
		layer string
		files []string
	}{layer: layer, files: currentSourceFiles})
	return int64(0), nil
}

func (m *mockStore) GetLayerHashes(ctx context.Context, layer string) (map[string]string, error) {
	if m.layerHashes == nil {
		return map[string]string{}, nil
	}
	if hashes, ok := m.layerHashes[layer]; ok {
		return hashes, nil
	}
	return map[string]string{}, nil
}

func (m *mockStore) FindEntityLayer(ctx context.Context, name string, layers []string) (string, error) {
	if m.entityLayers == nil {
		return "", nil
	}
	key := strings.ToLower(name)
	available := m.entityLayers[key]
	if available == nil {
		return "", nil
	}
	for _, layer := range layers {
		if _, ok := available[layer]; ok {
			return layer, nil
		}
	}
	return "", nil
}

func (m *mockStore) GetEntity(ctx context.Context, name, entityType string) (*store.Entity, error) {
	return nil, nil
}

func (m *mockStore) GetRelationships(ctx context.Context, name, relType, direction string, depth int) ([]store.Relationship, error) {
	return nil, nil
}

func (m *mockStore) ListEntities(ctx context.Context, entityType, layer, tag string) ([]store.EntitySummary, error) {
	return nil, nil
}

func (m *mockStore) ListEntitiesWithProperties(ctx context.Context) ([]store.Entity, error) {
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
	return nil, nil
}

func (m *mockStore) ListOrphanedEntities(ctx context.Context) ([]store.EntitySummary, error) {
	return nil, nil
}

func (m *mockStore) ListCrossLayerViolations(ctx context.Context) ([]store.EntitySummary, error) {
	return nil, nil
}

func (m *mockStore) RunSQL(ctx context.Context, query string, params map[string]any) ([]map[string]any, error) {
	return nil, nil
}

func TestRun_BasicIngestion(t *testing.T) {
	cfg := testProjectConfig(t)
	schema := testSchema(t)
	client := &mockStore{}

	result, err := Run(context.Background(), cfg, schema, client, Options{})
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	if !client.ensureCalled {
		t.Fatalf("expected ensure indexes")
	}
	if len(client.entities) != 3 {
		t.Fatalf("expected 3 entities, got %d", len(client.entities))
	}
	if len(client.relationships) == 0 {
		t.Fatalf("expected relationships")
	}
	if result.NodesUpserted != 3 {
		t.Fatalf("expected 3 nodes upserted, got %d", result.NodesUpserted)
	}
}

func TestRun_SkipsUnknownTypes(t *testing.T) {
	cfg := testProjectConfig(t)
	schema := testSchema(t)
	client := &mockStore{}

	result, err := Run(context.Background(), cfg, schema, client, Options{})
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	if result.FilesSkipped == 0 {
		t.Fatalf("expected files skipped")
	}
}

func TestRun_SkipsNoFrontmatter(t *testing.T) {
	cfg := testProjectConfig(t)
	schema := testSchema(t)
	client := &mockStore{}

	result, err := Run(context.Background(), cfg, schema, client, Options{})
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	if result.FilesSkipped == 0 {
		t.Fatalf("expected files skipped")
	}
}

func TestRun_RelatedField(t *testing.T) {
	cfg := testProjectConfig(t)
	schema := testSchema(t)
	client := &mockStore{}

	_, err := Run(context.Background(), cfg, schema, client, Options{})
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	relatedCount := 0
	for _, rel := range client.relationships {
		if rel.relType == "RELATED_TO" {
			relatedCount++
		}
	}
	if relatedCount != 2 {
		t.Fatalf("expected 2 RELATED_TO edges, got %d", relatedCount)
	}
}

func TestRun_FieldMappingResolution(t *testing.T) {
	cfg := testProjectConfig(t)
	schema := testSchema(t)
	client := &mockStore{}

	_, err := Run(context.Background(), cfg, schema, client, Options{})
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	found := false
	for _, rel := range client.relationships {
		if rel.relType == "MEMBER_OF" && rel.fromName == "Test NPC" && rel.toName == "The Watch" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected MEMBER_OF relationship")
	}
}

func TestRun_ContinuesOnError(t *testing.T) {
	cfg := testProjectConfig(t)
	schema := testSchema(t)
	client := &mockStore{failUpsert: true}

	result, err := Run(context.Background(), cfg, schema, client, Options{})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(result.Errors) == 0 {
		t.Fatalf("expected errors")
	}
}

func TestRun_RemoveStaleNodes(t *testing.T) {
	cfg := testProjectConfig(t)
	schema := testSchema(t)
	client := &mockStore{}

	_, err := Run(context.Background(), cfg, schema, client, Options{})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(client.removeCalls) != 1 {
		t.Fatalf("expected remove stale nodes call")
	}
	if len(client.removeCalls[0].files) == 0 {
		t.Fatalf("expected file list")
	}
}

func TestRun_IncrementalSkip(t *testing.T) {
	cfg := testProjectConfig(t)
	schema := testSchema(t)
	path := filepath.Join("testdata", "lore", "valid_npc.md")
	hash, err := computeHash(path)
	if err != nil {
		t.Fatalf("compute hash: %v", err)
	}
	client := &mockStore{
		layerHashes: map[string]map[string]string{
			"setting": {path: hash},
		},
	}

	_, err = Run(context.Background(), cfg, schema, client, Options{})
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	for _, entity := range client.entities {
		if entity.Name == "Test NPC" {
			t.Fatalf("expected Test NPC to be skipped")
		}
	}
	for _, rel := range client.relationships {
		if rel.fromName == "Test NPC" {
			t.Fatalf("expected relationships from Test NPC to be skipped")
		}
	}
}

func TestRun_EventConsequences(t *testing.T) {
	cfg := testProjectConfig(t)
	schema := testSchema(t)
	client := &mockStore{}

	_, err := Run(context.Background(), cfg, schema, client, Options{Full: true})
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	var found *store.EntityInput
	for i := range client.entities {
		if client.entities[i].Name == "Test Event" {
			found = &client.entities[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("expected Test Event entity")
	}

	payload, ok := found.Properties["consequences_json"].(string)
	if !ok || payload == "" {
		t.Fatalf("expected consequences_json property")
	}

	var got []Consequence
	if err := json.Unmarshal([]byte(payload), &got); err != nil {
		t.Fatalf("unmarshal consequences: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 consequences, got %d", len(got))
	}
	if got[0].Entity != "Test NPC" || got[0].Property != "status" || got[0].Value != "missing" {
		t.Fatalf("unexpected consequence 0: %#v", got[0])
	}
	if got[1].Entity != "The Watch" || got[1].Property != "members" || got[1].Add != "Test NPC" {
		t.Fatalf("unexpected consequence 1: %#v", got[1])
	}
}

func TestRun_FullIngestionOverridesHashes(t *testing.T) {
	cfg := testProjectConfig(t)
	schema := testSchema(t)
	path := filepath.Join("testdata", "lore", "valid_npc.md")
	hash, err := computeHash(path)
	if err != nil {
		t.Fatalf("compute hash: %v", err)
	}
	client := &mockStore{
		layerHashes: map[string]map[string]string{
			"setting": {path: hash},
		},
	}

	_, err = Run(context.Background(), cfg, schema, client, Options{Full: true})
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	found := false
	for _, entity := range client.entities {
		if entity.Name == "Test NPC" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected Test NPC to be ingested in full mode")
	}
}

func TestRun_DependsOnResolution(t *testing.T) {
	settingDir := t.TempDir()
	cfg := &config.ProjectConfig{
		Project:  "test",
		Version:  1,
		Database: config.DatabaseConfig{DSN: "postgres://localhost:5432/lorecraft"},
		Layers: []config.Layer{
			{Name: "setting", Paths: []string{settingDir}, Canonical: true},
			{Name: "campaign", Paths: []string{filepath.Join("testdata", "lore")}, Canonical: false, DependsOn: []string{"setting"}},
		},
	}
	schema := testSchema(t)
	client := &mockStore{
		entityLayers: map[string]map[string]struct{}{
			strings.ToLower("The Watch"): {"setting": {}},
		},
	}

	_, err := Run(context.Background(), cfg, schema, client, Options{Full: true})
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	found := false
	for _, rel := range client.relationships {
		if rel.relType == "MEMBER_OF" && rel.fromName == "Test NPC" {
			if rel.toLayer != "setting" {
				t.Fatalf("expected toLayer setting, got %q", rel.toLayer)
			}
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected MEMBER_OF relationship")
	}
}

func TestResolveFieldValue(t *testing.T) {
	cases := []struct {
		name     string
		value    any
		expected []string
	}{
		{name: "string", value: "A", expected: []string{"A"}},
		{name: "list", value: []any{"A", "B"}, expected: []string{"A", "B"}},
		{name: "nil", value: nil, expected: []string{}},
		{name: "integer", value: 42, expected: []string{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveFieldValue(tc.value)
			if !reflect.DeepEqual(got, tc.expected) {
				t.Fatalf("expected %#v, got %#v", tc.expected, got)
			}
		})
	}
}

func testProjectConfig(t *testing.T) *config.ProjectConfig {
	t.Helper()
	return &config.ProjectConfig{
		Project:  "test",
		Version:  1,
		Database: config.DatabaseConfig{DSN: "postgres://localhost:5432/lorecraft"},
		Layers: []config.Layer{{
			Name:  "setting",
			Paths: []string{filepath.Join("testdata", "lore")},
		}},
	}
}

func testSchema(t *testing.T) *config.Schema {
	t.Helper()
	schema, err := config.LoadSchema(filepath.Join("testdata", "schema.yaml"))
	if err != nil {
		t.Fatalf("load schema: %v", err)
	}
	return schema
}
