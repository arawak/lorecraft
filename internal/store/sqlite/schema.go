package sqlite

import (
	"context"
	"fmt"
	"strings"

	"lorecraft/internal/config"
)

func (c *Client) EnsureSchema(ctx context.Context, schema *config.Schema) error {
	ddl := `
	CREATE TABLE IF NOT EXISTS entities (
		id              INTEGER PRIMARY KEY AUTOINCREMENT,
		name            TEXT NOT NULL,
		name_normalized TEXT NOT NULL,
		entity_type     TEXT NOT NULL,
		layer           TEXT NOT NULL,
		source_file     TEXT,
		source_hash     TEXT,
		tags            TEXT DEFAULT '[]',
		properties      TEXT DEFAULT '{}',
		body            TEXT DEFAULT '',
		is_placeholder  INTEGER DEFAULT 0,
		last_ingested   TEXT DEFAULT (datetime('now')),
		CONSTRAINT uq_entity_name_layer UNIQUE (name_normalized, layer)
	);

	CREATE TABLE IF NOT EXISTS edges (
		id       INTEGER PRIMARY KEY AUTOINCREMENT,
		src_id   INTEGER NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
		dst_id   INTEGER NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
		rel_type TEXT NOT NULL,
		CONSTRAINT uq_edge UNIQUE (src_id, dst_id, rel_type)
	);

	CREATE TABLE IF NOT EXISTS events (
		id            INTEGER PRIMARY KEY AUTOINCREMENT,
		entity_id     INTEGER NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
		layer         TEXT NOT NULL,
		session       INTEGER NOT NULL,
		date_in_world TEXT DEFAULT '',
		consequences  TEXT DEFAULT '[]',
		CONSTRAINT uq_event_entity UNIQUE (entity_id)
	);

	CREATE INDEX IF NOT EXISTS idx_entities_layer ON entities (layer);
	CREATE INDEX IF NOT EXISTS idx_entities_type ON entities (entity_type);
	CREATE INDEX IF NOT EXISTS idx_entities_source_file ON entities (source_file);
	CREATE INDEX IF NOT EXISTS idx_entities_type_layer ON entities (entity_type, layer);
	CREATE INDEX IF NOT EXISTS idx_entities_name_norm ON entities (name_normalized);
	CREATE INDEX IF NOT EXISTS idx_entities_placeholder ON entities (is_placeholder) WHERE is_placeholder = 1;
	CREATE INDEX IF NOT EXISTS idx_edges_src ON edges (src_id);
	CREATE INDEX IF NOT EXISTS idx_edges_dst ON edges (dst_id);
	CREATE INDEX IF NOT EXISTS idx_edges_type ON edges (rel_type);
	CREATE INDEX IF NOT EXISTS idx_edges_src_type ON edges (src_id, rel_type);
	CREATE INDEX IF NOT EXISTS idx_edges_dst_type ON edges (dst_id, rel_type);
	CREATE INDEX IF NOT EXISTS idx_events_layer ON events (layer);
	CREATE INDEX IF NOT EXISTS idx_events_layer_session ON events (layer, session);

	CREATE VIRTUAL TABLE IF NOT EXISTS entities_fts USING fts5(
		name,
		tags,
		body,
		content=entities,
		content_rowid=id
	);

	CREATE TRIGGER IF NOT EXISTS entities_ai AFTER INSERT ON entities BEGIN
		INSERT INTO entities_fts(rowid, name, tags, body)
		VALUES (new.id, new.name, new.tags, new.body);
	END;

	CREATE TRIGGER IF NOT EXISTS entities_ad AFTER DELETE ON entities BEGIN
		INSERT INTO entities_fts(entities_fts, rowid, name, tags, body)
		VALUES ('delete', old.id, old.name, old.tags, old.body);
	END;

	CREATE TRIGGER IF NOT EXISTS entities_au AFTER UPDATE ON entities BEGIN
		INSERT INTO entities_fts(entities_fts, rowid, name, tags, body)
		VALUES ('delete', old.id, old.name, old.tags, old.body);
		INSERT INTO entities_fts(rowid, name, tags, body)
		VALUES (new.id, new.name, new.tags, new.body);
	END;
	`

	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	statements := splitStatements(ddl)
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("executing DDL: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing schema transaction: %w", err)
	}

	return nil
}

func splitStatements(ddl string) []string {
	var statements []string
	var current strings.Builder

	for _, line := range strings.Split(ddl, "\n") {
		stripped := strings.TrimSpace(line)
		if strings.HasPrefix(stripped, "--") {
			continue
		}
		current.WriteString(line)
		current.WriteString("\n")

		if strings.HasSuffix(stripped, ";") {
			statements = append(statements, current.String())
			current.Reset()
		}
	}

	if current.Len() > 0 {
		statements = append(statements, current.String())
	}

	return statements
}
