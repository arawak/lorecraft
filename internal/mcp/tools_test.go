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
	currentStateResult  *graph.CurrentState
	currentStateErr     error
	timelineResult      []graph.Event
	timelineErr         error

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
	lastTimelineLayer      string
	lastTimelineEntity     string
	lastTimelineFrom       int
	lastTimelineTo         int
	lastCurrentStateName   string
	lastCurrentStateLayer  string
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

func (m *mockGraphQuerier) GetCurrentState(ctx context.Context, name, layer string) (*graph.CurrentState, error) {
	m.lastCurrentStateName = name
	m.lastCurrentStateLayer = layer
	return m.currentStateResult, m.currentStateErr
}

func (m *mockGraphQuerier) GetTimeline(ctx context.Context, layer, entity string, fromSession, toSession int) ([]graph.Event, error) {
	m.lastTimelineLayer = layer
	m.lastTimelineEntity = entity
	m.lastTimelineFrom = fromSession
	m.lastTimelineTo = toSession
	return m.timelineResult, m.timelineErr
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
	if len(output.EntityTypes[0].FieldMappings) != 1 {
		t.Fatalf("unexpected field mappings output: %+v", output.EntityTypes[0].FieldMappings)
	}
	if output.EntityTypes[0].FieldMappings[0].TargetType == nil {
		t.Fatalf("expected empty target_type slice, got nil")
	}
}

func TestGetCurrentState(t *testing.T) {
	graphMock := &mockGraphQuerier{
		currentStateResult: &graph.CurrentState{
			BaseProperties:    map[string]any{"status": "intact"},
			CurrentProperties: map[string]any{"status": "damaged"},
			Events: []graph.Event{{
				Name:    "Storm Surge",
				Layer:   "campaign",
				Session: 1,
				Consequences: []graph.Consequence{{
					Entity:   "Westport",
					Property: "status",
					Value:    "damaged",
				}},
			}},
		},
	}
	server := NewServer(&config.Schema{Version: 1}, graphMock)

	_, output, err := server.handleGetCurrentState(context.Background(), nil, GetCurrentStateInput{Name: "Westport", Layer: "campaign"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output.CurrentProperties["status"] != "damaged" {
		t.Fatalf("unexpected current state output: %+v", output)
	}
	if graphMock.lastCurrentStateName != "Westport" || graphMock.lastCurrentStateLayer != "campaign" {
		t.Fatalf("unexpected current state params")
	}
}

func TestGetTimeline(t *testing.T) {
	graphMock := &mockGraphQuerier{
		timelineResult: []graph.Event{{Name: "Storm Surge", Layer: "campaign", Session: 1}},
	}
	server := NewServer(&config.Schema{Version: 1}, graphMock)

	_, output, err := server.handleGetTimeline(context.Background(), nil, GetTimelineInput{Layer: "campaign", Entity: "Westport", FromSession: 1, ToSession: 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(output.Events) != 1 || output.Events[0].Name != "Storm Surge" {
		t.Fatalf("unexpected timeline output: %+v", output)
	}
	if graphMock.lastTimelineLayer != "campaign" || graphMock.lastTimelineEntity != "Westport" || graphMock.lastTimelineFrom != 1 || graphMock.lastTimelineTo != 2 {
		t.Fatalf("unexpected timeline params")
	}
}

func TestCheckConsistency(t *testing.T) {
	graphMock := &mockGraphQuerier{
		entityResult: &graph.Entity{Name: "Westport", EntityType: "settlement", Layer: "setting"},
		relationshipsResult: []graph.Relationship{{
			From: graph.EntityRef{Name: "Westport", EntityType: "settlement", Layer: "setting"},
			To:   graph.EntityRef{Name: "The Westlands", EntityType: "region", Layer: "setting"},
			Type: "PART_OF",
		}},
		timelineResult: []graph.Event{{Name: "Storm Surge", Layer: "campaign", Session: 1}},
	}
	server := NewServer(&config.Schema{Version: 1}, graphMock)

	_, output, err := server.handleCheckConsistency(context.Background(), nil, CheckConsistencyInput{Name: "Westport", Layer: "campaign", Depth: 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output.Entity.Name != "Westport" {
		t.Fatalf("unexpected entity output: %+v", output.Entity)
	}
	if len(output.Relationships) != 1 || output.Relationships[0].Type != "PART_OF" {
		t.Fatalf("unexpected relationships output: %+v", output.Relationships)
	}
	if len(output.Events) != 1 || output.Events[0].Name != "Storm Surge" {
		t.Fatalf("unexpected events output: %+v", output.Events)
	}
}
