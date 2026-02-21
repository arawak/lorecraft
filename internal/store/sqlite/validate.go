package sqlite

import (
	"context"
	"encoding/json"

	"lorecraft/internal/store"
)

func (c *Client) ListDanglingPlaceholders(ctx context.Context) ([]store.EntitySummary, error) {
	query := `SELECT name, entity_type, layer, tags FROM entities WHERE is_placeholder = 1`

	rows, err := c.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []store.EntitySummary
	for rows.Next() {
		var s store.EntitySummary
		var tagsBytes []byte
		if err := rows.Scan(&s.Name, &s.EntityType, &s.Layer, &tagsBytes); err != nil {
			return nil, err
		}
		if len(tagsBytes) > 0 {
			if err := json.Unmarshal(tagsBytes, &s.Tags); err != nil {
				return nil, err
			}
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
	  AND e.is_placeholder = 0
	`

	rows, err := c.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []store.EntitySummary
	for rows.Next() {
		var s store.EntitySummary
		var tagsBytes []byte
		if err := rows.Scan(&s.Name, &s.EntityType, &s.Layer, &tagsBytes); err != nil {
			return nil, err
		}
		if len(tagsBytes) > 0 {
			if err := json.Unmarshal(tagsBytes, &s.Tags); err != nil {
				return nil, err
			}
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
	return []store.EntitySummary{}, nil
}
