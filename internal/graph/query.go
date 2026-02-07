package graph

import (
	"context"
	"fmt"
	"strings"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type Entity struct {
	Name       string
	EntityType string
	Layer      string
	SourceFile string
	SourceHash string
	Tags       []string
	Properties map[string]any
}

type EntitySummary struct {
	Name       string
	EntityType string
	Layer      string
	Tags       []string
}

type EntityRef struct {
	Name       string
	EntityType string
	Layer      string
}

type Relationship struct {
	From      EntityRef
	To        EntityRef
	Type      string
	Direction string
	Depth     int
}

type SearchResult struct {
	Name       string
	EntityType string
	Layer      string
	Tags       []string
	Score      float64
}

var standardPropertyKeys = map[string]struct{}{
	"name":            {},
	"name_normalized": {},
	"entity_type":     {},
	"layer":           {},
	"source_file":     {},
	"source_hash":     {},
	"last_ingested":   {},
	"tags":            {},
	"tags_text":       {},
	"_placeholder":    {},
}

func (c *Client) GetEntity(ctx context.Context, name, entityType string) (*Entity, error) {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: c.database})
	defer session.Close(ctx)

	query := "MATCH (n:Entity) WHERE n.name_normalized = $name_normalized"
	params := map[string]any{"name_normalized": strings.ToLower(name)}
	if strings.TrimSpace(entityType) != "" {
		query += " AND n.entity_type = $entity_type"
		params["entity_type"] = entityType
	}
	query += " RETURN n"

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, query, params)
		if err != nil {
			return nil, err
		}
		var nodes []neo4j.Node
		for res.Next(ctx) {
			value, _ := res.Record().Get("n")
			if node, ok := value.(neo4j.Node); ok {
				nodes = append(nodes, node)
			}
		}
		if err := res.Err(); err != nil {
			return nil, err
		}
		return nodes, nil
	})
	if err != nil {
		return nil, fmt.Errorf("getting entity: %w", err)
	}

	nodes := result.([]neo4j.Node)
	if len(nodes) == 0 {
		return nil, nil
	}
	if len(nodes) > 1 {
		return nil, fmt.Errorf("entity %q matched %d nodes", name, len(nodes))
	}

	node := nodes[0]
	props := node.Props
	entity := &Entity{
		Name:       toString(props["name"]),
		EntityType: toString(props["entity_type"]),
		Layer:      toString(props["layer"]),
		SourceFile: toString(props["source_file"]),
		SourceHash: toString(props["source_hash"]),
		Tags:       toStringSlice(props["tags"]),
		Properties: extractProperties(props),
	}
	return entity, nil
}

func (c *Client) ListEntities(ctx context.Context, entityType, layer, tag string) ([]EntitySummary, error) {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: c.database})
	defer session.Close(ctx)

	query := "MATCH (n:Entity)"
	params := map[string]any{}
	var conditions []string

	if strings.TrimSpace(entityType) != "" {
		conditions = append(conditions, "n.entity_type = $entity_type")
		params["entity_type"] = entityType
	}
	if strings.TrimSpace(layer) != "" {
		conditions = append(conditions, "n.layer = $layer")
		params["layer"] = layer
	}
	if strings.TrimSpace(tag) != "" {
		conditions = append(conditions, "$tag IN n.tags")
		params["tag"] = tag
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " RETURN n ORDER BY n.name"

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, query, params)
		if err != nil {
			return nil, err
		}
		var summaries []EntitySummary
		for res.Next(ctx) {
			value, _ := res.Record().Get("n")
			node, ok := value.(neo4j.Node)
			if !ok {
				continue
			}
			props := node.Props
			summaries = append(summaries, EntitySummary{
				Name:       toString(props["name"]),
				EntityType: toString(props["entity_type"]),
				Layer:      toString(props["layer"]),
				Tags:       toStringSlice(props["tags"]),
			})
		}
		if err := res.Err(); err != nil {
			return nil, err
		}
		return summaries, nil
	})
	if err != nil {
		return nil, fmt.Errorf("listing entities: %w", err)
	}

	return result.([]EntitySummary), nil
}

func (c *Client) GetRelationships(ctx context.Context, name, relType, direction string, depth int) ([]Relationship, error) {
	direction = strings.TrimSpace(direction)
	if direction == "" {
		direction = "both"
	}
	switch direction {
	case "outgoing", "incoming", "both":
	default:
		return nil, fmt.Errorf("invalid direction: %s", direction)
	}
	if depth < 1 || depth > 5 {
		return nil, fmt.Errorf("depth must be between 1 and 5")
	}
	if strings.TrimSpace(relType) != "" && !labelPattern.MatchString(relType) {
		return nil, fmt.Errorf("invalid relationship type: %s", relType)
	}

	session := c.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: c.database})
	defer session.Close(ctx)

	relPattern := ""
	if strings.TrimSpace(relType) != "" {
		relPattern = ":" + relType
	}

	var match string
	switch direction {
	case "outgoing":
		match = fmt.Sprintf("MATCH p=(start)-[%s*1..%d]->(n)", relPattern, depth)
	case "incoming":
		match = fmt.Sprintf("MATCH p=(start)<-[%s*1..%d]-(n)", relPattern, depth)
	case "both":
		match = fmt.Sprintf("MATCH p=(start)-[%s*1..%d]-(n)", relPattern, depth)
	}

	query := fmt.Sprintf(`
MATCH (start:Entity {name_normalized: $name_normalized})
%s
WITH p, nodes(p) AS ns, relationships(p) AS rs
UNWIND range(0, size(rs) - 1) AS idx
WITH ns[idx] AS hopFrom, ns[idx+1] AS hopTo, rs[idx] AS rel, idx
RETURN hopFrom, hopTo, type(rel) AS rel_type, startNode(rel) AS rel_start, idx
`, match)

	params := map[string]any{
		"name_normalized": strings.ToLower(name),
	}

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, query, params)
		if err != nil {
			return nil, err
		}
		var relationships []Relationship
		for res.Next(ctx) {
			record := res.Record()
			hopFrom, _ := record.Get("hopFrom")
			hopTo, _ := record.Get("hopTo")
			relTypeValue, _ := record.Get("rel_type")
			relStart, _ := record.Get("rel_start")
			idxValue, _ := record.Get("idx")

			fromNode, okFrom := hopFrom.(neo4j.Node)
			toNode, okTo := hopTo.(neo4j.Node)
			relStartNode, okRelStart := relStart.(neo4j.Node)
			depthValue, _ := idxValue.(int64)
			if !okFrom || !okTo {
				continue
			}

			var relDirection string
			if okRelStart && relStartNode.Id == fromNode.Id {
				relDirection = "outgoing"
			} else {
				relDirection = "incoming"
			}

			relationships = append(relationships, Relationship{
				From:      entityRefFromNode(fromNode),
				To:        entityRefFromNode(toNode),
				Type:      toString(relTypeValue),
				Direction: relDirection,
				Depth:     int(depthValue) + 1,
			})
		}
		if err := res.Err(); err != nil {
			return nil, err
		}
		return relationships, nil
	})
	if err != nil {
		return nil, fmt.Errorf("getting relationships: %w", err)
	}

	return result.([]Relationship), nil
}

func (c *Client) Search(ctx context.Context, query, layer, entityType string) ([]SearchResult, error) {
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("query must not be empty")
	}

	session := c.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: c.database})
	defer session.Close(ctx)

	cypher := `
CALL db.index.fulltext.queryNodes("entity_fulltext", $query) YIELD node, score
WHERE ($layer = "" OR node.layer = $layer)
  AND ($type = "" OR node.entity_type = $type)
RETURN node, score
ORDER BY score DESC, node.name
LIMIT $limit
`

	params := map[string]any{
		"query": query,
		"layer": layer,
		"type":  entityType,
		"limit": 50,
	}

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, cypher, params)
		if err != nil {
			return nil, err
		}
		var results []SearchResult
		for res.Next(ctx) {
			record := res.Record()
			nodeValue, _ := record.Get("node")
			scoreValue, _ := record.Get("score")
			node, ok := nodeValue.(neo4j.Node)
			if !ok {
				continue
			}
			props := node.Props
			score, _ := scoreValue.(float64)
			results = append(results, SearchResult{
				Name:       toString(props["name"]),
				EntityType: toString(props["entity_type"]),
				Layer:      toString(props["layer"]),
				Tags:       toStringSlice(props["tags"]),
				Score:      score,
			})
		}
		if err := res.Err(); err != nil {
			return nil, err
		}
		return results, nil
	})
	if err != nil {
		return nil, fmt.Errorf("searching entities: %w", err)
	}

	return result.([]SearchResult), nil
}

func toStringSlice(value any) []string {
	switch v := value.(type) {
	case nil:
		return nil
	case []string:
		return append([]string{}, v...)
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func extractProperties(props map[string]any) map[string]any {
	if len(props) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(props))
	for key, value := range props {
		if _, ok := standardPropertyKeys[key]; ok {
			continue
		}
		out[key] = value
	}
	return out
}

func toString(value any) string {
	if s, ok := value.(string); ok {
		return s
	}
	return ""
}

func entityRefFromNode(node neo4j.Node) EntityRef {
	props := node.Props
	return EntityRef{
		Name:       toString(props["name"]),
		EntityType: toString(props["entity_type"]),
		Layer:      toString(props["layer"]),
	}
}
