package postgres

import (
	"context"

	"lorecraft/internal/store"
)

func (c *Client) ListDanglingPlaceholders(ctx context.Context) ([]store.EntitySummary, error) {
	query := `SELECT name, entity_type, layer, tags FROM entities WHERE is_placeholder = TRUE`

	rows, err := c.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []store.EntitySummary
	for rows.Next() {
		var s store.EntitySummary
		if err := rows.Scan(&s.Name, &s.EntityType, &s.Layer, &s.Tags); err != nil {
			return nil, err
		}
		if s.Tags == nil {
			s.Tags = []string{}
		}
		summaries = append(summaries, s)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if summaries == nil {
		summaries = []store.EntitySummary{}
	}
	return summaries, nil
}

func (c *Client) ListOrphanedEntities(ctx context.Context) ([]store.EntitySummary, error) {
	query := `
SELECT e.name, e.entity_type, e.layer, e.tags FROM entities e
WHERE NOT EXISTS (SELECT 1 FROM edges WHERE src_id = e.id OR dst_id = e.id)
  AND e.is_placeholder = FALSE
`

	rows, err := c.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []store.EntitySummary
	for rows.Next() {
		var s store.EntitySummary
		if err := rows.Scan(&s.Name, &s.EntityType, &s.Layer, &s.Tags); err != nil {
			return nil, err
		}
		if s.Tags == nil {
			s.Tags = []string{}
		}
		summaries = append(summaries, s)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if summaries == nil {
		summaries = []store.EntitySummary{}
	}
	return summaries, nil
}

func (c *Client) ListCrossLayerViolations(ctx context.Context) ([]store.EntitySummary, error) {
	// TODO: Implement cross-layer violation detection once event/campaign layer logic is finalized
	return []store.EntitySummary{}, nil
}
