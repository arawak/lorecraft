package postgres

import (
	"context"
	"fmt"
	"strconv"
)

func (c *Client) RunSQL(ctx context.Context, query string, params map[string]any) ([]map[string]any, error) {
	args := make([]any, 0, len(params))
	for i := 1; i <= len(params); i++ {
		key := strconv.Itoa(i)
		if val, ok := params[key]; ok {
			args = append(args, val)
		}
	}

	rows, err := c.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("running sql: %w", err)
	}
	defer rows.Close()

	fieldDescriptions := rows.FieldDescriptions()
	results := make([]map[string]any, 0)

	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, fmt.Errorf("getting row values: %w", err)
		}

		row := make(map[string]any, len(fieldDescriptions))
		for i, fd := range fieldDescriptions {
			row[string(fd.Name)] = values[i]
		}
		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating sql rows: %w", err)
	}

	return results, nil
}
