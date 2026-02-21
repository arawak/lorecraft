package postgres

import (
	"context"
	"fmt"
	"strings"
)

func (c *Client) RemoveStaleNodes(ctx context.Context, layer string, currentSourceFiles []string) (int64, error) {
	query := `
DELETE FROM entities
WHERE layer = $1
  AND source_file IS NOT NULL
  AND source_file <> ''
  AND NOT (source_file = ANY($2))
  AND is_placeholder = FALSE
RETURNING id
`

	rows, err := c.pool.Query(ctx, query, layer, currentSourceFiles)
	if err != nil {
		return 0, fmt.Errorf("removing stale nodes: %w", err)
	}
	defer rows.Close()

	var count int64
	for rows.Next() {
		count++
	}

	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("counting deleted rows: %w", err)
	}

	return count, nil
}

func (c *Client) GetLayerHashes(ctx context.Context, layer string) (map[string]string, error) {
	query := `
SELECT source_file, source_hash FROM entities
WHERE layer = $1
  AND source_file IS NOT NULL
  AND source_file <> ''
  AND is_placeholder = FALSE
`

	rows, err := c.pool.Query(ctx, query, layer)
	if err != nil {
		return nil, fmt.Errorf("query layer hashes: %w", err)
	}
	defer rows.Close()

	hashes := make(map[string]string)
	for rows.Next() {
		var sourceFile, sourceHash string
		if err := rows.Scan(&sourceFile, &sourceHash); err != nil {
			return nil, fmt.Errorf("scanning layer hash: %w", err)
		}
		hashes[sourceFile] = sourceHash
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating layer hashes: %w", err)
	}

	return hashes, nil
}

func (c *Client) FindEntityLayer(ctx context.Context, name string, layers []string) (string, error) {
	if len(layers) == 0 {
		return "", nil
	}

	nameNormalized := strings.ToLower(name)
	for _, layer := range layers {
		var found string
		err := c.pool.QueryRow(ctx,
			"SELECT layer FROM entities WHERE name_normalized = $1 AND layer = $2 LIMIT 1",
			nameNormalized, layer,
		).Scan(&found)
		if err == nil {
			return found, nil
		}
	}

	return "", nil
}
