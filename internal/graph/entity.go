package graph

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

var labelPattern = regexp.MustCompile(`^[A-Z0-9_]+$`)

type EntityInput struct {
	Name       string
	EntityType string
	Label      string
	Layer      string
	SourceFile string
	SourceHash string
	Properties map[string]any
	Tags       []string
}

func (c *Client) UpsertEntity(ctx context.Context, e EntityInput) error {
	if strings.TrimSpace(e.Label) == "" || !labelPattern.MatchString(e.Label) {
		return fmt.Errorf("invalid label: %s", e.Label)
	}

	session := c.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: c.database})
	defer session.Close(ctx)

	query := fmt.Sprintf(`
MERGE (n:Entity {name_normalized: $name_normalized, layer: $layer})
SET n.name = $name,
    n.entity_type = $entity_type,
    n.source_file = $source_file,
    n.source_hash = $source_hash,
    n.last_ingested = datetime(),
    n.tags = $tags,
    n.tags_text = $tags_text,
    n:%s
REMOVE n:_Placeholder
SET n += $props
`, e.Label)

	params := map[string]any{
		"name_normalized": strings.ToLower(e.Name),
		"layer":           e.Layer,
		"name":            e.Name,
		"entity_type":     e.EntityType,
		"source_file":     e.SourceFile,
		"source_hash":     e.SourceHash,
		"tags":            e.Tags,
		"tags_text":       strings.Join(e.Tags, " "),
		"props":           e.Properties,
	}

	if _, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		_, err := tx.Run(ctx, query, params)
		return nil, err
	}); err != nil {
		return fmt.Errorf("upserting entity: %w", err)
	}

	return nil
}
