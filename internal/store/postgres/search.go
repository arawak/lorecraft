package postgres

import (
	"context"
	"fmt"
	"strings"

	"lorecraft/internal/store"
)

func (c *Client) Search(ctx context.Context, query, layer, entityType string) ([]store.SearchResult, error) {
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("query must not be empty")
	}

	sql := `
SELECT name, entity_type, layer, tags,
    ts_rank(search_vector, websearch_to_tsquery('english', $1)) AS score,
    CASE WHEN body <> '' THEN
        ts_headline('english', body, websearch_to_tsquery('english', $1),
            'MaxFragments=2, MaxWords=40, MinWords=20, StartSel=**, StopSel=**')
    ELSE '' END AS snippet
FROM entities
WHERE search_vector @@ websearch_to_tsquery('english', $1)
  AND ($2 = '' OR layer = $2)
  AND ($3 = '' OR entity_type = $3)
  AND is_placeholder = FALSE
ORDER BY score DESC, name ASC
LIMIT 50
`

	rows, err := c.pool.Query(ctx, sql, query, layer, entityType)
	if err != nil {
		return nil, fmt.Errorf("searching entities: %w", err)
	}
	defer rows.Close()

	var results []store.SearchResult
	for rows.Next() {
		var r store.SearchResult
		err := rows.Scan(&r.Name, &r.EntityType, &r.Layer, &r.Tags, &r.Score, &r.Snippet)
		if err != nil {
			return nil, fmt.Errorf("scanning search result: %w", err)
		}
		if r.Tags == nil {
			r.Tags = []string{}
		}
		results = append(results, r)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating search results: %w", err)
	}

	if results == nil {
		results = []store.SearchResult{}
	}

	return results, nil
}
