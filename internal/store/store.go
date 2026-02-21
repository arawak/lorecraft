package store

import (
	"context"

	"lorecraft/internal/config"
)

type Store interface {
	Close(ctx context.Context) error
	EnsureSchema(ctx context.Context, schema *config.Schema) error

	UpsertEntity(ctx context.Context, e EntityInput) error
	UpsertRelationship(ctx context.Context, fromName, fromLayer, toName, toLayer, relType string) error
	RemoveStaleNodes(ctx context.Context, layer string, currentSourceFiles []string) (int64, error)
	GetLayerHashes(ctx context.Context, layer string) (map[string]string, error)
	FindEntityLayer(ctx context.Context, name string, layers []string) (string, error)

	GetEntity(ctx context.Context, name, entityType string) (*Entity, error)
	GetRelationships(ctx context.Context, name, relType, direction string, depth int) ([]Relationship, error)
	ListEntities(ctx context.Context, entityType, layer, tag string) ([]EntitySummary, error)
	ListEntitiesWithProperties(ctx context.Context) ([]Entity, error)
	Search(ctx context.Context, query, layer, entityType string) ([]SearchResult, error)
	GetCurrentState(ctx context.Context, name, layer string) (*CurrentState, error)
	GetTimeline(ctx context.Context, layer, entity string, fromSession, toSession int) ([]Event, error)

	ListDanglingPlaceholders(ctx context.Context) ([]EntitySummary, error)
	ListOrphanedEntities(ctx context.Context) ([]EntitySummary, error)
	ListCrossLayerViolations(ctx context.Context) ([]EntitySummary, error)

	RunSQL(ctx context.Context, query string, params map[string]any) ([]map[string]any, error)
}
