package postgres

import (
	"context"
	"fmt"

	"lorecraft/internal/config"
)

func (c *Client) EnsureSchema(ctx context.Context, schema *config.Schema) error {
	// Note: All DDL statements are executed in a single call, which PostgreSQL
	// runs atomically within an implicit transaction. The use of "IF NOT EXISTS"
	// makes this idempotent for the initial schema creation and subsequent runs
	// without destructive schema changes. However, as the schema evolves with
	// more migrations, consider implementing a dedicated migration table and
	// tool (e.g., migrate, flyway) to track schema versions and enable
	// non-idempotent migrations (e.g., column renames, data transformations).
	ddl := `
CREATE TABLE IF NOT EXISTS entities (
    id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name            TEXT NOT NULL,
    name_normalized TEXT NOT NULL,
    entity_type     TEXT NOT NULL,
    layer           TEXT NOT NULL,
    source_file     TEXT,
    source_hash     TEXT,
    tags            TEXT[] DEFAULT '{}',
    properties      JSONB DEFAULT '{}',
    body            TEXT DEFAULT '',
    is_placeholder  BOOLEAN DEFAULT FALSE,
    last_ingested   TIMESTAMPTZ DEFAULT now(),
    CONSTRAINT uq_entity_name_layer UNIQUE (name_normalized, layer)
);

ALTER TABLE entities ADD COLUMN IF NOT EXISTS search_vector TSVECTOR;

CREATE TABLE IF NOT EXISTS edges (
    id       BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    src_id   BIGINT NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    dst_id   BIGINT NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    rel_type TEXT NOT NULL,
    CONSTRAINT uq_edge UNIQUE (src_id, dst_id, rel_type)
);

CREATE TABLE IF NOT EXISTS events (
    id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    entity_id     BIGINT NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    layer         TEXT NOT NULL,
    session       INTEGER NOT NULL,
    date_in_world TEXT DEFAULT '',
    consequences  JSONB DEFAULT '[]',
    CONSTRAINT uq_event_entity UNIQUE (entity_id)
);

CREATE INDEX IF NOT EXISTS idx_entities_search ON entities USING GIN (search_vector);
CREATE INDEX IF NOT EXISTS idx_entities_layer ON entities (layer);
CREATE INDEX IF NOT EXISTS idx_entities_type ON entities (entity_type);
CREATE INDEX IF NOT EXISTS idx_entities_source_file ON entities (source_file);
CREATE INDEX IF NOT EXISTS idx_entities_type_layer ON entities (entity_type, layer);
CREATE INDEX IF NOT EXISTS idx_entities_name_norm ON entities (name_normalized);
CREATE INDEX IF NOT EXISTS idx_entities_placeholder ON entities (is_placeholder) WHERE is_placeholder = TRUE;
CREATE INDEX IF NOT EXISTS idx_entities_tags ON entities USING GIN (tags);
CREATE INDEX IF NOT EXISTS idx_edges_src ON edges (src_id);
CREATE INDEX IF NOT EXISTS idx_edges_dst ON edges (dst_id);
CREATE INDEX IF NOT EXISTS idx_edges_type ON edges (rel_type);
CREATE INDEX IF NOT EXISTS idx_edges_src_type ON edges (src_id, rel_type);
CREATE INDEX IF NOT EXISTS idx_edges_dst_type ON edges (dst_id, rel_type);
CREATE INDEX IF NOT EXISTS idx_events_layer ON events (layer);
CREATE INDEX IF NOT EXISTS idx_events_layer_session ON events (layer, session);
`
	_, err := c.pool.Exec(ctx, ddl)
	if err != nil {
		return fmt.Errorf("ensuring schema: %w", err)
	}
	return nil
}
