package mcp

import (
	"context"
	"fmt"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"lorecraft/internal/config"
	"lorecraft/internal/graph"
)

type SearchLoreInput struct {
	Query string `json:"query" jsonschema:"search terms"`
	Layer string `json:"layer,omitempty" jsonschema:"restrict to a specific layer"`
	Type  string `json:"type,omitempty" jsonschema:"restrict to a specific entity type"`
}

type GetEntityInput struct {
	Name string `json:"name" jsonschema:"entity name"`
	Type string `json:"type,omitempty" jsonschema:"optional entity type"`
}

type GetRelationshipsInput struct {
	Name      string `json:"name" jsonschema:"starting entity name"`
	Type      string `json:"type,omitempty" jsonschema:"relationship type filter"`
	Depth     int    `json:"depth,omitempty" jsonschema:"maximum traversal depth"`
	Direction string `json:"direction,omitempty" jsonschema:"outgoing, incoming, or both"`
}

type ListEntitiesInput struct {
	Type  string `json:"type,omitempty" jsonschema:"entity type filter"`
	Layer string `json:"layer,omitempty" jsonschema:"layer filter"`
	Tag   string `json:"tag,omitempty" jsonschema:"tag filter"`
}

type GetSchemaInput struct{}

type EntityOutput struct {
	Name       string         `json:"name"`
	EntityType string         `json:"type"`
	Layer      string         `json:"layer"`
	SourceFile string         `json:"source_file"`
	SourceHash string         `json:"source_hash"`
	Tags       []string       `json:"tags"`
	Properties map[string]any `json:"properties"`
}

type EntitySummaryOutput struct {
	Name       string   `json:"name"`
	EntityType string   `json:"type"`
	Layer      string   `json:"layer"`
	Tags       []string `json:"tags"`
}

type RelationshipOutput struct {
	From      EntityRefOutput `json:"from"`
	To        EntityRefOutput `json:"to"`
	Type      string          `json:"type"`
	Direction string          `json:"direction"`
	Depth     int             `json:"depth"`
}

type EntityRefOutput struct {
	Name       string `json:"name"`
	EntityType string `json:"type"`
	Layer      string `json:"layer"`
}

type SearchResultOutput struct {
	Name       string   `json:"name"`
	EntityType string   `json:"type"`
	Layer      string   `json:"layer"`
	Tags       []string `json:"tags"`
	Score      float64  `json:"score"`
}

type SearchLoreOutput struct {
	Results []SearchResultOutput `json:"results"`
}

type SchemaOutput struct {
	Version           int                      `json:"version"`
	EntityTypes       []EntityTypeOutput       `json:"entity_types"`
	RelationshipTypes []RelationshipTypeOutput `json:"relationship_types"`
}

type EntityTypeOutput struct {
	Name          string               `json:"name"`
	Properties    []PropertyOutput     `json:"properties"`
	FieldMappings []FieldMappingOutput `json:"field_mappings"`
}

type PropertyOutput struct {
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	Values  []string `json:"values,omitempty"`
	Default string   `json:"default,omitempty"`
}

type FieldMappingOutput struct {
	Field        string   `json:"field"`
	Relationship string   `json:"relationship"`
	TargetType   []string `json:"target_type"`
}

type RelationshipTypeOutput struct {
	Name      string `json:"name"`
	Inverse   string `json:"inverse,omitempty"`
	Symmetric bool   `json:"symmetric,omitempty"`
}

type GetRelationshipsOutput struct {
	Relationships []RelationshipOutput `json:"relationships"`
}

type ListEntitiesOutput struct {
	Entities []EntitySummaryOutput `json:"entities"`
}

func (s *Server) registerTools() {
	sdk.AddTool(s.mcp, &sdk.Tool{
		Name:        "search_lore",
		Description: "Search entities by name, tags, and text",
	}, s.handleSearchLore)

	sdk.AddTool(s.mcp, &sdk.Tool{
		Name:        "get_entity",
		Description: "Retrieve a specific entity and its properties",
	}, s.handleGetEntity)

	sdk.AddTool(s.mcp, &sdk.Tool{
		Name:        "get_relationships",
		Description: "Traverse relationships from an entity",
	}, s.handleGetRelationships)

	sdk.AddTool(s.mcp, &sdk.Tool{
		Name:        "list_entities",
		Description: "List entities with optional filters",
	}, s.handleListEntities)

	sdk.AddTool(s.mcp, &sdk.Tool{
		Name:        "get_schema",
		Description: "Return the current schema definition",
	}, s.handleGetSchema)
}

func (s *Server) handleSearchLore(ctx context.Context, req *sdk.CallToolRequest, input SearchLoreInput) (*sdk.CallToolResult, SearchLoreOutput, error) {
	if input.Query == "" {
		return nil, SearchLoreOutput{}, fmt.Errorf("query is required")
	}
	results, err := s.graph.Search(ctx, input.Query, input.Layer, input.Type)
	if err != nil {
		return nil, SearchLoreOutput{}, err
	}

	output := make([]SearchResultOutput, 0, len(results))
	for _, result := range results {
		output = append(output, searchResultOutputFromGraph(result))
	}
	return nil, SearchLoreOutput{Results: output}, nil
}

func (s *Server) handleGetEntity(ctx context.Context, req *sdk.CallToolRequest, input GetEntityInput) (*sdk.CallToolResult, EntityOutput, error) {
	if input.Name == "" {
		return nil, EntityOutput{}, fmt.Errorf("name is required")
	}
	entity, err := s.graph.GetEntity(ctx, input.Name, input.Type)
	if err != nil {
		return nil, EntityOutput{}, err
	}
	if entity == nil {
		return nil, EntityOutput{}, fmt.Errorf("entity not found")
	}
	return nil, entityOutputFromGraph(entity), nil
}

func (s *Server) handleGetRelationships(ctx context.Context, req *sdk.CallToolRequest, input GetRelationshipsInput) (*sdk.CallToolResult, GetRelationshipsOutput, error) {
	if input.Name == "" {
		return nil, GetRelationshipsOutput{}, fmt.Errorf("name is required")
	}
	depth := input.Depth
	if depth == 0 {
		depth = 1
	}
	rels, err := s.graph.GetRelationships(ctx, input.Name, input.Type, input.Direction, depth)
	if err != nil {
		return nil, GetRelationshipsOutput{}, err
	}

	output := make([]RelationshipOutput, 0, len(rels))
	for _, rel := range rels {
		output = append(output, relationshipOutputFromGraph(rel))
	}
	return nil, GetRelationshipsOutput{Relationships: output}, nil
}

func (s *Server) handleListEntities(ctx context.Context, req *sdk.CallToolRequest, input ListEntitiesInput) (*sdk.CallToolResult, ListEntitiesOutput, error) {
	items, err := s.graph.ListEntities(ctx, input.Type, input.Layer, input.Tag)
	if err != nil {
		return nil, ListEntitiesOutput{}, err
	}

	output := make([]EntitySummaryOutput, 0, len(items))
	for _, item := range items {
		output = append(output, entitySummaryOutputFromGraph(item))
	}
	return nil, ListEntitiesOutput{Entities: output}, nil
}

func (s *Server) handleGetSchema(ctx context.Context, req *sdk.CallToolRequest, input GetSchemaInput) (*sdk.CallToolResult, SchemaOutput, error) {
	return nil, schemaOutputFromConfig(s.schema), nil
}

func schemaOutputFromConfig(schema *config.Schema) SchemaOutput {
	if schema == nil {
		return SchemaOutput{}
	}

	out := SchemaOutput{
		Version:           schema.Version,
		EntityTypes:       make([]EntityTypeOutput, 0, len(schema.EntityTypes)),
		RelationshipTypes: make([]RelationshipTypeOutput, 0, len(schema.RelationshipTypes)),
	}

	for _, entityType := range schema.EntityTypes {
		entityOut := EntityTypeOutput{
			Name:          entityType.Name,
			Properties:    make([]PropertyOutput, 0, len(entityType.Properties)),
			FieldMappings: make([]FieldMappingOutput, 0, len(entityType.FieldMappings)),
		}
		for _, prop := range entityType.Properties {
			entityOut.Properties = append(entityOut.Properties, PropertyOutput{
				Name:    prop.Name,
				Type:    prop.Type,
				Values:  prop.Values,
				Default: prop.Default,
			})
		}
		for _, mapping := range entityType.FieldMappings {
			entityOut.FieldMappings = append(entityOut.FieldMappings, FieldMappingOutput{
				Field:        mapping.Field,
				Relationship: mapping.Relationship,
				TargetType:   mapping.TargetType,
			})
		}
		out.EntityTypes = append(out.EntityTypes, entityOut)
	}

	for _, rel := range schema.RelationshipTypes {
		out.RelationshipTypes = append(out.RelationshipTypes, RelationshipTypeOutput{
			Name:      rel.Name,
			Inverse:   rel.Inverse,
			Symmetric: rel.Symmetric,
		})
	}

	return out
}

func entityOutputFromGraph(entity *graph.Entity) EntityOutput {
	if entity == nil {
		return EntityOutput{}
	}
	properties := map[string]any{}
	for key, value := range entity.Properties {
		properties[key] = value
	}
	return EntityOutput{
		Name:       entity.Name,
		EntityType: entity.EntityType,
		Layer:      entity.Layer,
		SourceFile: entity.SourceFile,
		SourceHash: entity.SourceHash,
		Tags:       append([]string{}, entity.Tags...),
		Properties: properties,
	}
}

func entitySummaryOutputFromGraph(entity graph.EntitySummary) EntitySummaryOutput {
	return EntitySummaryOutput{
		Name:       entity.Name,
		EntityType: entity.EntityType,
		Layer:      entity.Layer,
		Tags:       append([]string{}, entity.Tags...),
	}
}

func searchResultOutputFromGraph(result graph.SearchResult) SearchResultOutput {
	return SearchResultOutput{
		Name:       result.Name,
		EntityType: result.EntityType,
		Layer:      result.Layer,
		Tags:       append([]string{}, result.Tags...),
		Score:      result.Score,
	}
}

func relationshipOutputFromGraph(rel graph.Relationship) RelationshipOutput {
	return RelationshipOutput{
		From: EntityRefOutput{
			Name:       rel.From.Name,
			EntityType: rel.From.EntityType,
			Layer:      rel.From.Layer,
		},
		To: EntityRefOutput{
			Name:       rel.To.Name,
			EntityType: rel.To.EntityType,
			Layer:      rel.To.Layer,
		},
		Type:      rel.Type,
		Direction: rel.Direction,
		Depth:     rel.Depth,
	}
}
