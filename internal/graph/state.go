package graph

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"

	"lorecraft/internal/config"
)

type Consequence struct {
	Entity   string `json:"entity"`
	Property string `json:"property"`
	Value    any    `json:"value,omitempty"`
	Add      any    `json:"add,omitempty"`
}

type Event struct {
	Name         string
	Layer        string
	Session      int
	DateInWorld  string
	Participants []string
	Location     []string
	Consequences []Consequence
}

type CurrentState struct {
	BaseProperties    map[string]any
	Events            []Event
	CurrentProperties map[string]any
}

func (c *Client) GetCurrentState(ctx context.Context, name, layer string) (*CurrentState, error) {
	if strings.TrimSpace(layer) == "" {
		return nil, fmt.Errorf("layer is required")
	}

	baseLayer, err := resolveCanonicalLayer(layer)
	if err != nil {
		return nil, err
	}

	baseProps, err := c.fetchEntityProperties(ctx, name, baseLayer)
	if err != nil {
		return nil, err
	}
	if baseProps == nil {
		return nil, nil
	}

	events, err := c.fetchEventsForEntity(ctx, name, layer)
	if err != nil {
		return nil, err
	}

	current := copyProperties(baseProps)
	for _, event := range events {
		applyConsequences(current, event.Consequences, name)
	}

	return &CurrentState{
		BaseProperties:    baseProps,
		Events:            events,
		CurrentProperties: current,
	}, nil
}

func (c *Client) GetTimeline(ctx context.Context, layer, entity string, fromSession, toSession int) ([]Event, error) {
	if strings.TrimSpace(layer) == "" {
		return nil, fmt.Errorf("layer is required")
	}

	session := c.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: c.database})
	defer session.Close(ctx)

	entityNormalized := strings.ToLower(strings.TrimSpace(entity))
	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, `
MATCH (e:Entity {entity_type: "event", layer: $layer})
WHERE ($entity = "" OR EXISTS {
  MATCH (e)-[:AFFECTS|:INVOLVES]->(n:Entity {name_normalized: $entity})
})
  AND ($from = 0 OR e.session >= $from)
  AND ($to = 0 OR e.session <= $to)
OPTIONAL MATCH (e)-[:INVOLVES]->(p:Entity)
OPTIONAL MATCH (e)-[:OCCURS_IN]->(l:Entity)
WITH e, collect(DISTINCT p.name) AS participants, collect(DISTINCT l.name) AS locations
RETURN e, participants, locations
ORDER BY e.session ASC`, map[string]any{
			"layer":  layer,
			"entity": entityNormalized,
			"from":   fromSession,
			"to":     toSession,
		})
		if err != nil {
			return nil, err
		}
		var events []Event
		for res.Next(ctx) {
			record := res.Record()
			value, _ := record.Get("e")
			participants, _ := record.Get("participants")
			locations, _ := record.Get("locations")
			node, ok := value.(neo4j.Node)
			if !ok {
				continue
			}
			event, err := eventFromNode(node, participants, locations)
			if err != nil {
				return nil, err
			}
			events = append(events, event)
		}
		if err := res.Err(); err != nil {
			return nil, err
		}
		return events, nil
	})
	if err != nil {
		return nil, fmt.Errorf("get timeline: %w", err)
	}

	return result.([]Event), nil
}

func (c *Client) fetchEntityProperties(ctx context.Context, name, layer string) (map[string]any, error) {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: c.database})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, "MATCH (n:Entity {name_normalized: $name, layer: $layer}) RETURN n", map[string]any{
			"name":  strings.ToLower(name),
			"layer": layer,
		})
		if err != nil {
			return nil, err
		}
		if res.Next(ctx) {
			value, _ := res.Record().Get("n")
			node, ok := value.(neo4j.Node)
			if ok {
				return extractProperties(node.Props), nil
			}
		}
		if err := res.Err(); err != nil {
			return nil, err
		}
		return nil, nil
	})
	if err != nil {
		return nil, fmt.Errorf("fetching base entity: %w", err)
	}

	if result == nil {
		return nil, nil
	}
	return result.(map[string]any), nil
}

func (c *Client) fetchEventsForEntity(ctx context.Context, name, layer string) ([]Event, error) {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: c.database})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, `
MATCH (e:Entity {entity_type: "event", layer: $layer})-[:AFFECTS]->(n:Entity {name_normalized: $name})
OPTIONAL MATCH (e)-[:INVOLVES]->(p:Entity)
OPTIONAL MATCH (e)-[:OCCURS_IN]->(l:Entity)
WITH e, collect(DISTINCT p.name) AS participants, collect(DISTINCT l.name) AS locations
RETURN e, participants, locations
ORDER BY e.session ASC`, map[string]any{
			"layer": layer,
			"name":  strings.ToLower(name),
		})
		if err != nil {
			return nil, err
		}
		var events []Event
		for res.Next(ctx) {
			record := res.Record()
			value, _ := record.Get("e")
			participants, _ := record.Get("participants")
			locations, _ := record.Get("locations")
			node, ok := value.(neo4j.Node)
			if !ok {
				continue
			}
			event, err := eventFromNode(node, participants, locations)
			if err != nil {
				return nil, err
			}
			events = append(events, event)
		}
		if err := res.Err(); err != nil {
			return nil, err
		}
		return events, nil
	})
	if err != nil {
		return nil, fmt.Errorf("fetching events: %w", err)
	}

	return result.([]Event), nil
}

func eventFromNode(node neo4j.Node, participants any, locations any) (Event, error) {
	props := node.Props
	consequences, err := decodeConsequences(props["consequences_json"])
	if err != nil {
		return Event{}, err
	}

	return Event{
		Name:         toString(props["name"]),
		Layer:        toString(props["layer"]),
		Session:      toInt(props["session"]),
		DateInWorld:  toString(props["date_in_world"]),
		Participants: toStringSlice(participants),
		Location:     toStringSlice(locations),
		Consequences: consequences,
	}, nil
}

func decodeConsequences(value any) ([]Consequence, error) {
	if value == nil {
		return nil, nil
	}
	payload, ok := value.(string)
	if !ok || payload == "" {
		return nil, nil
	}
	var consequences []Consequence
	if err := json.Unmarshal([]byte(payload), &consequences); err != nil {
		return nil, fmt.Errorf("decode consequences: %w", err)
	}
	return consequences, nil
}

func applyConsequences(props map[string]any, consequences []Consequence, target string) {
	nameNormalized := strings.ToLower(strings.TrimSpace(target))
	for _, consequence := range consequences {
		if nameNormalized != "" && strings.ToLower(consequence.Entity) != nameNormalized {
			continue
		}
		if consequence.Value != nil {
			props[consequence.Property] = consequence.Value
			continue
		}
		if consequence.Add != nil {
			props[consequence.Property] = appendValue(props[consequence.Property], consequence.Add)
		}
	}
}

func appendValue(existing any, add any) any {
	switch current := existing.(type) {
	case []string:
		if value, ok := add.(string); ok {
			return append(current, value)
		}
		out := make([]any, 0, len(current)+1)
		for _, item := range current {
			out = append(out, item)
		}
		return append(out, add)
	case []any:
		return append(current, add)
	default:
		return []any{add}
	}
}

func copyProperties(props map[string]any) map[string]any {
	if props == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(props))
	for key, value := range props {
		out[key] = value
	}
	return out
}

func toInt(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return 0
	}
}

func resolveCanonicalLayer(layerName string) (string, error) {
	cfg, err := config.LoadProjectConfig("lorecraft.yaml")
	if err != nil {
		return "", fmt.Errorf("load project config: %w", err)
	}

	byName := make(map[string]config.Layer)
	for _, layer := range cfg.Layers {
		byName[strings.ToLower(layer.Name)] = layer
	}

	start, ok := byName[strings.ToLower(layerName)]
	if !ok {
		return "", fmt.Errorf("unknown layer: %s", layerName)
	}
	if start.Canonical {
		return start.Name, nil
	}

	var visit func(layer config.Layer) (string, bool)
	visit = func(layer config.Layer) (string, bool) {
		for _, dep := range layer.DependsOn {
			depLayer, ok := byName[strings.ToLower(dep)]
			if !ok {
				continue
			}
			if depLayer.Canonical {
				return depLayer.Name, true
			}
			if name, found := visit(depLayer); found {
				return name, true
			}
		}
		return "", false
	}

	if name, found := visit(start); found {
		return name, nil
	}

	return "", fmt.Errorf("no canonical base layer found for %s", layerName)
}
