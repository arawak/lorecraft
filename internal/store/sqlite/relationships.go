package sqlite

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

	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	var srcID int64
	err = tx.QueryRowContext(ctx,
		"SELECT id FROM entities WHERE name_normalized = ? AND layer = ?",
		strings.ToLower(fromName), fromLayer,
	).Scan(&srcID)
	if err != nil {
		return fmt.Errorf("finding source entity: %w", err)
	}

	var dstID int64
	err = tx.QueryRowContext(ctx,
		`INSERT INTO entities (name, name_normalized, entity_type, layer, is_placeholder, tags, properties)
		VALUES (?, ?, '', ?, 1, '[]', '{}')
		ON CONFLICT (name_normalized, layer) DO UPDATE SET name = entities.name
		RETURNING id`,
		toName, strings.ToLower(toName), toLayer,
	).Scan(&dstID)
	if err != nil {
		return fmt.Errorf("upserting target entity: %w", err)
	}

	_, err = tx.ExecContext(ctx,
		`INSERT OR IGNORE INTO edges (src_id, dst_id, rel_type) VALUES (?, ?, ?)`,
		srcID, dstID, relType,
	)
	if err != nil {
		return fmt.Errorf("upserting edge: %w", err)
	}

	if err := tx.Commit(); err != nil {
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

	var startID int64
	err := c.db.QueryRowContext(ctx,
		"SELECT id FROM entities WHERE name_normalized = ?",
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

		placeholders := make([]string, len(frontier))
		args := make([]any, len(frontier)+1)
		for i, id := range frontier {
			placeholders[i] = "?"
			args[i+1] = id
		}
		args[0] = strings.Join(placeholders, ",")

		var query string
		switch direction {
		case "outgoing":
			query = fmt.Sprintf(`
			SELECT e.src_id, e.dst_id, e.rel_type,
				   s.name AS src_name, s.entity_type AS src_type, s.layer AS src_layer,
				   d.name AS dst_name, d.entity_type AS dst_type, d.layer AS dst_layer
			FROM edges e
			JOIN entities s ON e.src_id = s.id
			JOIN entities d ON e.dst_id = d.id
			WHERE e.src_id IN (%s)
			  AND (? = '' OR e.rel_type = ?)`, args[0].(string))
		case "incoming":
			query = fmt.Sprintf(`
			SELECT e.src_id, e.dst_id, e.rel_type,
				   s.name AS src_name, s.entity_type AS src_type, s.layer AS src_layer,
				   d.name AS dst_name, d.entity_type AS dst_type, d.layer AS dst_layer
			FROM edges e
			JOIN entities s ON e.src_id = s.id
			JOIN entities d ON e.dst_id = d.id
			WHERE e.dst_id IN (%s)
			  AND (? = '' OR e.rel_type = ?)`, args[0].(string))
		case "both":
			query = fmt.Sprintf(`
			SELECT e.src_id, e.dst_id, e.rel_type,
				   s.name AS src_name, s.entity_type AS src_type, s.layer AS src_layer,
				   d.name AS dst_name, d.entity_type AS dst_type, d.layer AS dst_layer
			FROM edges e
			JOIN entities s ON e.src_id = s.id
			JOIN entities d ON e.dst_id = d.id
			WHERE (e.src_id IN (%s) OR e.dst_id IN (%s))
			  AND (? = '' OR e.rel_type = ?)`, args[0].(string), args[0].(string))
		}

		queryArgs := make([]any, 0)
		for _, id := range frontier {
			queryArgs = append(queryArgs, id)
		}
		if direction == "both" {
			for _, id := range frontier {
				queryArgs = append(queryArgs, id)
			}
		}
		queryArgs = append(queryArgs, relType, relType)

		rows, err := c.db.QueryContext(ctx, query, queryArgs...)
		if err != nil {
			return nil, fmt.Errorf("querying relationships: %w", err)
		}

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
				rows.Close()
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
		rows.Close()

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
