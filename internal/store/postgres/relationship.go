package postgres

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"lorecraft/internal/store"
)

var relTypePattern = regexp.MustCompile(`^[A-Z0-9_]+$`)

func (c *Client) UpsertRelationship(ctx context.Context, fromName, fromLayer, toName, toLayer, relType string) error {
	if strings.TrimSpace(relType) == "" || !relTypePattern.MatchString(relType) {
		return fmt.Errorf("invalid relationship type: %s", relType)
	}

	tx, err := c.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var srcID int64
	err = tx.QueryRow(ctx,
		"SELECT id FROM entities WHERE name_normalized = $1 AND layer = $2",
		strings.ToLower(fromName), fromLayer,
	).Scan(&srcID)
	if err != nil {
		return fmt.Errorf("finding source entity: %w", err)
	}

	var dstID int64
	err = tx.QueryRow(ctx,
		`INSERT INTO entities (name, name_normalized, entity_type, layer, is_placeholder)
VALUES ($1, $2, '', $3, TRUE)
ON CONFLICT (name_normalized, layer) DO UPDATE SET name = entities.name
RETURNING id`,
		toName, strings.ToLower(toName), toLayer,
	).Scan(&dstID)
	if err != nil {
		return fmt.Errorf("upserting target entity: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO edges (src_id, dst_id, rel_type) VALUES ($1, $2, $3)
ON CONFLICT (src_id, dst_id, rel_type) DO NOTHING`,
		srcID, dstID, relType,
	)
	if err != nil {
		return fmt.Errorf("upserting edge: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	return nil
}

func (c *Client) GetRelationships(ctx context.Context, name, relType, direction string, depth int) ([]store.Relationship, error) {
	direction = strings.TrimSpace(direction)
	if direction == "" {
		direction = "both"
	}
	switch direction {
	case "outgoing", "incoming", "both":
	default:
		return nil, fmt.Errorf("invalid direction: %s", direction)
	}
	if depth < 1 || depth > 5 {
		return nil, fmt.Errorf("depth must be between 1 and 5")
	}
	if strings.TrimSpace(relType) != "" && !relTypePattern.MatchString(relType) {
		return nil, fmt.Errorf("invalid relationship type: %s", relType)
	}

	// Note on direction semantics at depth > 1:
	// At depth 1, direction is unambiguous: edges from the starting node are "outgoing",
	// edges to the starting node are "incoming". At depth > 1, the frontier contains
	// intermediate nodes, and edges between frontier nodes may be traversed from either
	// direction. The direction assigned to multi-hop relationships is based on the first
	// frontier node matched in the result set, which may not match user intuition.
	// For "both" direction at depth > 1, consider results as undirected or use depth 1.

	var startID int64
	err := c.pool.QueryRow(ctx,
		"SELECT id FROM entities WHERE name_normalized = $1",
		strings.ToLower(name),
	).Scan(&startID)
	if err != nil {
		return nil, fmt.Errorf("finding start entity: %w", err)
	}

	visited := make(map[int64]bool)
	visited[startID] = true
	frontier := []int64{startID}
	var results []store.Relationship

	for currentDepth := 1; currentDepth <= depth; currentDepth++ {
		if len(frontier) == 0 {
			break
		}

		var query string
		switch direction {
		case "outgoing":
			query = `
SELECT e.src_id, e.dst_id, e.rel_type,
       s.name AS src_name, s.entity_type AS src_type, s.layer AS src_layer,
       d.name AS dst_name, d.entity_type AS dst_type, d.layer AS dst_layer
FROM edges e
JOIN entities s ON e.src_id = s.id
JOIN entities d ON e.dst_id = d.id
WHERE e.src_id = ANY($1)
  AND ($2 = '' OR e.rel_type = $2)`
		case "incoming":
			query = `
SELECT e.src_id, e.dst_id, e.rel_type,
       s.name AS src_name, s.entity_type AS src_type, s.layer AS src_layer,
       d.name AS dst_name, d.entity_type AS dst_type, d.layer AS dst_layer
FROM edges e
JOIN entities s ON e.src_id = s.id
JOIN entities d ON e.dst_id = d.id
WHERE e.dst_id = ANY($1)
  AND ($2 = '' OR e.rel_type = $2)`
		case "both":
			query = `
SELECT e.src_id, e.dst_id, e.rel_type,
       s.name AS src_name, s.entity_type AS src_type, s.layer AS src_layer,
       d.name AS dst_name, d.entity_type AS dst_type, d.layer AS dst_layer
FROM edges e
JOIN entities s ON e.src_id = s.id
JOIN entities d ON e.dst_id = d.id
WHERE (e.src_id = ANY($1) OR e.dst_id = ANY($1))
  AND ($2 = '' OR e.rel_type = $2)`
		}

		rows, err := c.pool.Query(ctx, query, frontier, relType)
		if err != nil {
			return nil, fmt.Errorf("querying relationships: %w", err)
		}
		defer rows.Close()

		var newFrontier []int64
		for rows.Next() {
			var srcID, dstID int64
			var rel store.Relationship
			var srcType, dstType string
			var srcLayer, dstLayer string

			err := rows.Scan(&srcID, &dstID, &rel.Type,
				&rel.From.Name, &srcType, &srcLayer,
				&rel.To.Name, &dstType, &dstLayer,
			)
			if err != nil {
				return nil, fmt.Errorf("scanning relationship: %w", err)
			}

			rel.From.EntityType = srcType
			rel.From.Layer = srcLayer
			rel.To.EntityType = dstType
			rel.To.Layer = dstLayer

			var otherID int64
			var isFromFrontier bool
			for _, fid := range frontier {
				if fid == srcID {
					otherID = dstID
					isFromFrontier = true
					break
				} else if fid == dstID {
					otherID = srcID
					isFromFrontier = false
					break
				}
			}

			if visited[otherID] {
				continue
			}

			if isFromFrontier {
				rel.Direction = "outgoing"
			} else {
				rel.Direction = "incoming"
				rel.From, rel.To = rel.To, rel.From
			}

			rel.Depth = currentDepth
			results = append(results, rel)
			newFrontier = append(newFrontier, otherID)
			visited[otherID] = true
		}

		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("iterating relationship rows: %w", err)
		}

		frontier = newFrontier
	}

	if results == nil {
		results = []store.Relationship{}
	}

	return results, nil
}
