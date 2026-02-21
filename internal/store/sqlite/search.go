package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"lorecraft/internal/store"
)

func (c *Client) Search(ctx context.Context, query, layer, entityType string) ([]store.SearchResult, error) {
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("query must not be empty")
	}

	ftsQuery := convertWebsearchToFTS5(query)

	sqlQuery := `
	SELECT e.name, e.entity_type, e.layer, e.tags,
		   bm25(entities_fts, 10.0, 4.0, 1.0) AS score,
		   snippet(entities_fts, 2, '**', '**', '...', 50) AS snippet
	FROM entities_fts
	JOIN entities e ON entities_fts.rowid = e.id
	WHERE entities_fts MATCH ?
	  AND (? = '' OR e.layer = ?)
	  AND (? = '' OR e.entity_type = ?)
	  AND e.is_placeholder = 0
	ORDER BY score DESC, e.name ASC
	LIMIT 50
	`

	rows, err := c.db.QueryContext(ctx, sqlQuery, ftsQuery, layer, layer, entityType, entityType)
	if err != nil {
		return nil, fmt.Errorf("searching entities: %w", err)
	}
	defer rows.Close()

	var results []store.SearchResult
	for rows.Next() {
		var r store.SearchResult
		var tagsBytes []byte
		err := rows.Scan(&r.Name, &r.EntityType, &r.Layer, &tagsBytes, &r.Score, &r.Snippet)
		if err != nil {
			return nil, fmt.Errorf("scanning search result: %w", err)
		}
		if len(tagsBytes) > 0 {
			if err := json.Unmarshal(tagsBytes, &r.Tags); err != nil {
				return nil, fmt.Errorf("unmarshaling tags: %w", err)
			}
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

func convertWebsearchToFTS5(query string) string {
	var result strings.Builder
	var inQuote bool
	var current strings.Builder

	flushToken := func() {
		token := current.String()
		current.Reset()
		if token == "" {
			return
		}

		upper := strings.ToUpper(token)
		switch upper {
		case "AND", "OR", "NOT":
			if result.Len() > 0 {
				result.WriteString(" ")
			}
			result.WriteString(upper)
			return
		}

		if result.Len() > 0 {
			lastWord := lastWord(result.String())
			if lastWord != "AND" && lastWord != "OR" && lastWord != "NOT" && lastWord != "" {
				result.WriteString(" AND ")
			} else {
				result.WriteString(" ")
			}
		}

		if strings.HasPrefix(token, "-") && len(token) > 1 {
			result.WriteString("NOT ")
			result.WriteString(token[1:])
		} else if strings.HasSuffix(token, "*") {
			result.WriteString(token)
		} else {
			result.WriteString(token)
		}
	}

	for i := 0; i < len(query); i++ {
		ch := query[i]
		switch {
		case ch == '"':
			if inQuote {
				inQuote = false
				token := current.String()
				current.Reset()
				if token != "" {
					if result.Len() > 0 {
						result.WriteString(" AND ")
					}
					result.WriteString(`"`)
					result.WriteString(token)
					result.WriteString(`"`)
				}
			} else {
				flushToken()
				inQuote = true
			}
		case inQuote:
			current.WriteByte(ch)
		case ch == ' ' || ch == '\t':
			flushToken()
		default:
			current.WriteByte(ch)
		}
	}

	flushToken()

	return result.String()
}

func lastWord(s string) string {
	words := strings.Fields(s)
	if len(words) == 0 {
		return ""
	}
	return words[len(words)-1]
}
