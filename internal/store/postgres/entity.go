package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"lorecraft/internal/store"
)

func (c *Client) UpsertEntity(ctx context.Context, e store.EntityInput) error {
	nameNormalized := strings.ToLower(e.Name)

	propsJSON, err := json.Marshal(e.Properties)
	if err != nil {
		return fmt.Errorf("marshaling properties: %w", err)
	}

	tags := e.Tags
	if len(tags) == 0 {
		tags = nil
	}

	query := `
INSERT INTO entities (name, name_normalized, entity_type, layer, source_file, source_hash, tags, properties, body, is_placeholder, last_ingested, search_vector)
VALUES ($1, $2, $3, $4, $5, $6, COALESCE($7, '{}'::text[]), $8, $9, FALSE, now(),
    setweight(to_tsvector('simple', coalesce($1, '')), 'A') ||
    setweight(to_tsvector('english', coalesce(array_to_string(COALESCE($7, '{}'::text[]), ' '), '')), 'B') ||
    setweight(to_tsvector('english', coalesce($9, '')), 'C')
)
ON CONFLICT (name_normalized, layer) DO UPDATE SET
    name = EXCLUDED.name,
    entity_type = EXCLUDED.entity_type,
    source_file = EXCLUDED.source_file,
    source_hash = EXCLUDED.source_hash,
    tags = EXCLUDED.tags,
    properties = EXCLUDED.properties,
    body = EXCLUDED.body,
    is_placeholder = FALSE,
    last_ingested = now(),
    search_vector = EXCLUDED.search_vector
`

	_, err = c.pool.Exec(ctx, query,
		e.Name,
		nameNormalized,
		e.EntityType,
		e.Layer,
		e.SourceFile,
		e.SourceHash,
		tags,
		propsJSON,
		e.Body,
	)
	if err != nil {
		return fmt.Errorf("upserting entity: %w", err)
	}
	return nil
}

func (c *Client) GetEntity(ctx context.Context, name, entityType string) (*store.Entity, error) {
	nameNormalized := strings.ToLower(name)

	query := `
SELECT id, name, entity_type, layer, source_file, source_hash, tags, properties, body
FROM entities
WHERE name_normalized = $1
  AND ($2 = '' OR entity_type = $2)
  AND is_placeholder = FALSE
`

	rows, err := c.pool.Query(ctx, query, nameNormalized, entityType)
	if err != nil {
		return nil, fmt.Errorf("getting entity: %w", err)
	}
	defer rows.Close()

	var entities []store.Entity
	for rows.Next() {
		var e store.Entity
		var propsBytes []byte
		err := rows.Scan(
			new(int64),
			&e.Name,
			&e.EntityType,
			&e.Layer,
			&e.SourceFile,
			&e.SourceHash,
			&e.Tags,
			&propsBytes,
			&e.Body,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning entity: %w", err)
		}
		if len(propsBytes) > 0 {
			if err := json.Unmarshal(propsBytes, &e.Properties); err != nil {
				return nil, fmt.Errorf("unmarshaling properties: %w", err)
			}
		}
		if e.Properties == nil {
			e.Properties = map[string]any{}
		}
		if e.Tags == nil {
			e.Tags = []string{}
		}
		entities = append(entities, e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating entity rows: %w", err)
	}

	if len(entities) == 0 {
		return nil, nil
	}
	if len(entities) > 1 {
		return nil, fmt.Errorf("internal error: entity uniqueness constraint violated (found %d rows for %q)", len(entities), name)
	}

	return &entities[0], nil
}

func (c *Client) ListEntities(ctx context.Context, entityType, layer, tag string) ([]store.EntitySummary, error) {
	query := `
SELECT name, entity_type, layer, tags
FROM entities
WHERE ($1 = '' OR entity_type = $1)
  AND ($2 = '' OR layer = $2)
  AND ($3 = '' OR $3 = ANY(tags))
  AND is_placeholder = FALSE
ORDER BY name
`

	rows, err := c.pool.Query(ctx, query, entityType, layer, tag)
	if err != nil {
		return nil, fmt.Errorf("listing entities: %w", err)
	}
	defer rows.Close()

	var summaries []store.EntitySummary
	for rows.Next() {
		var s store.EntitySummary
		err := rows.Scan(&s.Name, &s.EntityType, &s.Layer, &s.Tags)
		if err != nil {
			return nil, fmt.Errorf("scanning entity summary: %w", err)
		}
		if s.Tags == nil {
			s.Tags = []string{}
		}
		summaries = append(summaries, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating entity summaries: %w", err)
	}

	if summaries == nil {
		summaries = []store.EntitySummary{}
	}

	return summaries, nil
}

func (c *Client) ListEntitiesWithProperties(ctx context.Context) ([]store.Entity, error) {
	query := `
SELECT name, entity_type, layer, source_file, source_hash, tags, properties, body
FROM entities
WHERE is_placeholder = FALSE
ORDER BY name
`

	rows, err := c.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("listing entities with properties: %w", err)
	}
	defer rows.Close()

	var entities []store.Entity
	for rows.Next() {
		var e store.Entity
		var propsBytes []byte
		err := rows.Scan(
			&e.Name,
			&e.EntityType,
			&e.Layer,
			&e.SourceFile,
			&e.SourceHash,
			&e.Tags,
			&propsBytes,
			&e.Body,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning entity: %w", err)
		}
		if len(propsBytes) > 0 {
			if err := json.Unmarshal(propsBytes, &e.Properties); err != nil {
				return nil, fmt.Errorf("unmarshaling properties: %w", err)
			}
		}
		if e.Properties == nil {
			e.Properties = map[string]any{}
		}
		if e.Tags == nil {
			e.Tags = []string{}
		}
		entities = append(entities, e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating entities: %w", err)
	}

	if entities == nil {
		entities = []store.Entity{}
	}

	return entities, nil
}
