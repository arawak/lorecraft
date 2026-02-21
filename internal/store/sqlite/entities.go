package sqlite

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

	tagsJSON, err := json.Marshal(e.Tags)
	if err != nil {
		return fmt.Errorf("marshaling tags: %w", err)
	}

	query := `
	INSERT INTO entities (name, name_normalized, entity_type, layer, source_file, source_hash, tags, properties, body, is_placeholder, last_ingested)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 0, datetime('now'))
	ON CONFLICT (name_normalized, layer) DO UPDATE SET
		name = excluded.name,
		entity_type = excluded.entity_type,
		source_file = excluded.source_file,
		source_hash = excluded.source_hash,
		tags = excluded.tags,
		properties = excluded.properties,
		body = excluded.body,
		is_placeholder = 0,
		last_ingested = datetime('now')
	`

	_, err = c.db.ExecContext(ctx, query,
		e.Name,
		nameNormalized,
		e.EntityType,
		e.Layer,
		e.SourceFile,
		e.SourceHash,
		tagsJSON,
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
	WHERE name_normalized = ?
	  AND (? = '' OR entity_type = ?)
	  AND is_placeholder = 0
	`

	rows, err := c.db.QueryContext(ctx, query, nameNormalized, entityType, entityType)
	if err != nil {
		return nil, fmt.Errorf("getting entity: %w", err)
	}
	defer rows.Close()

	var entities []store.Entity
	for rows.Next() {
		var e store.Entity
		var propsBytes []byte
		var tagsBytes []byte
		err := rows.Scan(
			new(int64),
			&e.Name,
			&e.EntityType,
			&e.Layer,
			&e.SourceFile,
			&e.SourceHash,
			&tagsBytes,
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
		if len(tagsBytes) > 0 {
			if err := json.Unmarshal(tagsBytes, &e.Tags); err != nil {
				return nil, fmt.Errorf("unmarshaling tags: %w", err)
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
	WHERE (? = '' OR entity_type = ?)
	  AND (? = '' OR layer = ?)
	  AND is_placeholder = 0
	ORDER BY name
	`

	rows, err := c.db.QueryContext(ctx, query, entityType, entityType, layer, layer)
	if err != nil {
		return nil, fmt.Errorf("listing entities: %w", err)
	}
	defer rows.Close()

	var summaries []store.EntitySummary
	for rows.Next() {
		var s store.EntitySummary
		var tagsBytes []byte
		err := rows.Scan(&s.Name, &s.EntityType, &s.Layer, &tagsBytes)
		if err != nil {
			return nil, fmt.Errorf("scanning entity summary: %w", err)
		}
		if len(tagsBytes) > 0 {
			if err := json.Unmarshal(tagsBytes, &s.Tags); err != nil {
				return nil, fmt.Errorf("unmarshaling tags: %w", err)
			}
		}
		if s.Tags == nil {
			s.Tags = []string{}
		}

		if tag != "" && !containsTag(s.Tags, tag) {
			continue
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
	WHERE is_placeholder = 0
	ORDER BY name
	`

	rows, err := c.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("listing entities with properties: %w", err)
	}
	defer rows.Close()

	var entities []store.Entity
	for rows.Next() {
		var e store.Entity
		var propsBytes []byte
		var tagsBytes []byte
		err := rows.Scan(
			&e.Name,
			&e.EntityType,
			&e.Layer,
			&e.SourceFile,
			&e.SourceHash,
			&tagsBytes,
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
		if len(tagsBytes) > 0 {
			if err := json.Unmarshal(tagsBytes, &e.Tags); err != nil {
				return nil, fmt.Errorf("unmarshaling tags: %w", err)
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

func containsTag(tags []string, tag string) bool {
	for _, t := range tags {
		if strings.EqualFold(t, tag) {
			return true
		}
	}
	return false
}
