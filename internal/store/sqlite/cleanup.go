package sqlite

import (
	"context"
	"fmt"
	"strings"
)

func (c *Client) RemoveStaleNodes(ctx context.Context, layer string, currentSourceFiles []string) (int64, error) {
	if len(currentSourceFiles) == 0 {
		return 0, nil
	}

	placeholders := make([]string, len(currentSourceFiles))
	args := make([]any, len(currentSourceFiles)+1)
	args[0] = layer
	for i, f := range currentSourceFiles {
		placeholders[i] = "?"
		args[i+1] = f
	}

	query := fmt.Sprintf(`
	DELETE FROM entities
	WHERE layer = ?
	  AND source_file IS NOT NULL
	  AND source_file <> ''
	  AND source_file NOT IN (%s)
	  AND is_placeholder = 0
	`, strings.Join(placeholders, ", "))

	result, err := c.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("removing stale nodes: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("getting rows affected: %w", err)
	}

	return affected, nil
}

func (c *Client) GetLayerHashes(ctx context.Context, layer string) (map[string]string, error) {
	query := `
	SELECT source_file, source_hash FROM entities
	WHERE layer = ?
	  AND source_file IS NOT NULL
	  AND source_file <> ''
	  AND is_placeholder = 0
	`

	rows, err := c.db.QueryContext(ctx, query, layer)
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
		err := c.db.QueryRowContext(ctx,
			"SELECT layer FROM entities WHERE name_normalized = ? AND layer = ? LIMIT 1",
			nameNormalized, layer,
		).Scan(&found)
		if err == nil {
			return found, nil
		}
	}

	return "", nil
}
