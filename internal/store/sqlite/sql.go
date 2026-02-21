package sqlite

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

	rows, err := c.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("running sql: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("getting columns: %w", err)
	}

	results := make([]map[string]any, 0)

	for rows.Next() {
		values := make([]any, len(columns))
		valuePtrs := make([]any, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}

		row := make(map[string]any, len(columns))
		for i, col := range columns {
			row[col] = values[i]
		}
		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating sql rows: %w", err)
	}

	return results, nil
}
