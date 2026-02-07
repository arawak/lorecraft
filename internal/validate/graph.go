package validate

import (
	"context"

	"lorecraft/internal/graph"
)

type GraphValidator interface {
	ListEntities(ctx context.Context, entityType, layer, tag string) ([]graph.EntitySummary, error)
	GetEntity(ctx context.Context, name, entityType string) (*graph.Entity, error)
	ListDanglingPlaceholders(ctx context.Context) ([]graph.EntitySummary, error)
	ListOrphanedEntities(ctx context.Context) ([]graph.EntitySummary, error)
	ListDuplicateNames(ctx context.Context) ([]graph.EntitySummary, error)
	ListCrossLayerViolations(ctx context.Context) ([]graph.EntitySummary, error)
}
