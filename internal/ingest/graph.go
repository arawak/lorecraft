package ingest

import (
	"context"

	"lorecraft/internal/config"
	"lorecraft/internal/graph"
)

type GraphClient interface {
	UpsertEntity(ctx context.Context, e graph.EntityInput) error
	UpsertRelationship(ctx context.Context, fromName, fromLayer, toName, toLayer, relType string) error
	RemoveStaleNodes(ctx context.Context, layer string, currentSourceFiles []string) (int64, error)
	EnsureIndexes(ctx context.Context, schema *config.Schema) error
}
