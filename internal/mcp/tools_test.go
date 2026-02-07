package mcp

import (
	"context"
	"testing"

	"lorecraft/internal/config"
	"lorecraft/internal/graph"
)

type mockGraphQuerier struct {
	entityResult        *graph.Entity
	entityErr           error
	searchResult        []graph.SearchResult
	searchErr           error
	listResult          []graph.EntitySummary
	listErr             error
	relationshipsResult []graph.Relationship
	relationshipsErr    error

	lastGetEntityName      string
	lastGetEntityType      string
	lastSearchQuery        string
	lastSearchLayer        string
	lastSearchType         string
	lastListType           string
	lastListLayer          string
	lastListTag            string
	lastRelationshipsName  string
	lastRelationshipsType  string
	lastRelationshipsDir   string
	lastRelationshipsDepth int
}

func (m *mockGraphQuerier) GetEntity(ctx context.Context, name, entityType string) (*graph.Entity, error) {
	m.lastGetEntityName = name
	m.lastGetEntityType = entityType
	return m.entityResult, m.entityErr
}

func (m *mockGraphQuerier) GetRelationships(ctx context.Context, name, relType, direction string, depth int) ([]graph.Relationship, error) {
	m.lastRelationshipsName = name
	m.lastRelationshipsType = relType
	m.lastRelationshipsDir = direction
	m.lastRelationshipsDepth = depth
	return m.relationshipsResult, m.relationshipsErr
}

func (m *mockGraphQuerier) ListEntities(ctx context.Context, entityType, layer, tag string) ([]graph.EntitySummary, error) {
	m.lastListType = entityType
	m.lastListLayer = layer
	m.lastListTag = tag
	return m.listResult, m.listErr
}

func (m *mockGraphQuerier) Search(ctx context.Context, query, layer, entityType string) ([]graph.SearchResult, error) {
	m.lastSearchQuery = query
	m.lastSearchLayer = layer
	m.lastSearchType = entityType
	return m.searchResult, m.searchErr
}

func TestGetEntity_NotFound(t *testing.T) {
	server := NewServer(&config.Schema{Version: 1}, &mockGraphQuerier{})

	_, _, err := server.handleGetEntity(context.Background(), nil, GetEntityInput{Name: "Missing"})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestSearchLore(t *testing.T) {
	graphMock := &mockGraphQuerier{
		searchResult: []graph.SearchResult{
			{Name: "Westport", EntityType: "settlement", Layer: "setting", Tags: []string{"coastal"}, Score: 1.0},
		},
	}
	server := NewServer(&config.Schema{Version: 1}, graphMock)

	_, output, err := server.handleSearchLore(context.Background(), nil, SearchLoreInput{Query: "west", Layer: "setting", Type: "settlement"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(output.Results) != 1 || output.Results[0].Name != "Westport" {
		t.Fatalf("unexpected search output: %+v", output)
	}
	if graphMock.lastSearchQuery != "west" || graphMock.lastSearchLayer != "setting" || graphMock.lastSearchType != "settlement" {
		t.Fatalf("unexpected search params")
	}
}

func TestListEntities(t *testing.T) {
	graphMock := &mockGraphQuerier{
		listResult: []graph.EntitySummary{{Name: "A", EntityType: "npc", Layer: "setting", Tags: []string{"alpha"}}},
	}
	server := NewServer(&config.Schema{Version: 1}, graphMock)

	_, output, err := server.handleListEntities(context.Background(), nil, ListEntitiesInput{Type: "npc", Layer: "setting", Tag: "alpha"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(output.Entities) != 1 || output.Entities[0].Name != "A" {
		t.Fatalf("unexpected list output: %+v", output)
	}
	if graphMock.lastListType != "npc" || graphMock.lastListLayer != "setting" || graphMock.lastListTag != "alpha" {
		t.Fatalf("unexpected list params")
	}
}

func TestGetRelationships(t *testing.T) {
	graphMock := &mockGraphQuerier{
		relationshipsResult: []graph.Relationship{{
			From:      graph.EntityRef{Name: "A", EntityType: "npc", Layer: "setting"},
			To:        graph.EntityRef{Name: "B", EntityType: "npc", Layer: "setting"},
			Type:      "RELATED_TO",
			Direction: "outgoing",
			Depth:     1,
		}},
	}
	server := NewServer(&config.Schema{Version: 1}, graphMock)

	_, output, err := server.handleGetRelationships(context.Background(), nil, GetRelationshipsInput{Name: "A", Type: "RELATED_TO", Depth: 2, Direction: "both"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(output.Relationships) != 1 || output.Relationships[0].Type != "RELATED_TO" {
		t.Fatalf("unexpected relationships output: %+v", output)
	}
	if graphMock.lastRelationshipsName != "A" || graphMock.lastRelationshipsType != "RELATED_TO" || graphMock.lastRelationshipsDepth != 2 || graphMock.lastRelationshipsDir != "both" {
		t.Fatalf("unexpected relationships params")
	}
}

func TestGetSchema(t *testing.T) {
	schema := &config.Schema{
		Version: 1,
		EntityTypes: []config.EntityType{{
			Name: "npc",
			Properties: []config.Property{{
				Name: "role",
				Type: "string",
			}},
			FieldMappings: []config.FieldMapping{{
				Field:        "faction",
				Relationship: "MEMBER_OF",
			}},
		}},
		RelationshipTypes: []config.RelationshipType{{Name: "MEMBER_OF"}},
	}
	server := NewServer(schema, &mockGraphQuerier{})

	_, output, err := server.handleGetSchema(context.Background(), nil, GetSchemaInput{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output.Version != 1 || len(output.EntityTypes) != 1 {
		t.Fatalf("unexpected schema output: %+v", output)
	}
}
