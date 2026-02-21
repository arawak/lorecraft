package sqlite

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"lorecraft/internal/store"
)

func (c *Client) GetCurrentState(ctx context.Context, name, layer string) (*store.CurrentState, error) {
	if strings.TrimSpace(layer) == "" {
		return nil, fmt.Errorf("layer is required")
	}

	baseLayer, err := c.resolveCanonicalLayer(layer)
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

	return &store.CurrentState{
		BaseProperties:    baseProps,
		Events:            events,
		CurrentProperties: current,
	}, nil
}

func (c *Client) GetTimeline(ctx context.Context, layer, entity string, fromSession, toSession int) ([]store.Event, error) {
	if strings.TrimSpace(layer) == "" {
		return nil, fmt.Errorf("layer is required")
	}

	entityNormalized := strings.ToLower(strings.TrimSpace(entity))

	query := `
	SELECT e_ent.id, e_ent.name, ev.layer, ev.session, ev.date_in_world, ev.consequences
	FROM events ev
	JOIN entities e_ent ON ev.entity_id = e_ent.id
	WHERE ev.layer = ?
	  AND (? = '' OR EXISTS (
		  SELECT 1 FROM edges ea
		  JOIN entities t ON ea.dst_id = t.id
		  WHERE ea.src_id = e_ent.id
			AND ea.rel_type IN ('AFFECTS', 'INVOLVES')
			AND t.name_normalized = ?
	  ))
	  AND (? = 0 OR ev.session >= ?)
	  AND (? = 0 OR ev.session <= ?)
	ORDER BY ev.session ASC, ev.id ASC
	`

	rows, err := c.db.QueryContext(ctx, query, layer, entityNormalized, entityNormalized, fromSession, fromSession, toSession, toSession)
	if err != nil {
		return nil, fmt.Errorf("get timeline: %w", err)
	}
	defer rows.Close()

	var events []store.Event
	for rows.Next() {
		var event store.Event
		var entityID int64
		var consequencesBytes []byte

		err := rows.Scan(&entityID, &event.Name, &event.Layer, &event.Session, &event.DateInWorld, &consequencesBytes)
		if err != nil {
			return nil, fmt.Errorf("scanning event: %w", err)
		}

		if len(consequencesBytes) > 0 {
			if err := json.Unmarshal(consequencesBytes, &event.Consequences); err != nil {
				return nil, fmt.Errorf("unmarshaling consequences: %w", err)
			}
		}

		event.Participants, err = c.fetchEventParticipants(ctx, entityID)
		if err != nil {
			return nil, err
		}
		event.Location, err = c.fetchEventLocations(ctx, entityID)
		if err != nil {
			return nil, err
		}

		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating events: %w", err)
	}

	if events == nil {
		events = []store.Event{}
	}

	return events, nil
}

func (c *Client) fetchEntityProperties(ctx context.Context, name, layer string) (map[string]any, error) {
	query := `SELECT properties FROM entities WHERE name_normalized = ? AND layer = ?`

	var propsBytes []byte
	err := c.db.QueryRowContext(ctx, query, strings.ToLower(name), layer).Scan(&propsBytes)
	if err != nil {
		if errors.Is(err, errors.New("sql: no rows")) {
			return nil, nil
		}
		return nil, fmt.Errorf("fetching base entity: %w", err)
	}

	var props map[string]any
	if len(propsBytes) > 0 {
		if err := json.Unmarshal(propsBytes, &props); err != nil {
			return nil, fmt.Errorf("unmarshaling properties: %w", err)
		}
	}

	if props == nil {
		props = map[string]any{}
	}
	return props, nil
}

func (c *Client) fetchEventsForEntity(ctx context.Context, name, layer string) ([]store.Event, error) {
	query := `
	SELECT e_ent.id, e_ent.name, ev.layer, ev.session, ev.date_in_world, ev.consequences
	FROM events ev
	JOIN entities e_ent ON ev.entity_id = e_ent.id
	JOIN edges ed ON ed.src_id = e_ent.id AND ed.rel_type = 'AFFECTS'
	JOIN entities target ON ed.dst_id = target.id
	WHERE target.name_normalized = ? AND ev.layer = ?
	ORDER BY ev.session ASC, ev.id ASC
	`

	rows, err := c.db.QueryContext(ctx, query, strings.ToLower(name), layer)
	if err != nil {
		return nil, fmt.Errorf("fetching events: %w", err)
	}
	defer rows.Close()

	var events []store.Event
	for rows.Next() {
		var event store.Event
		var entityID int64
		var consequencesBytes []byte

		err := rows.Scan(&entityID, &event.Name, &event.Layer, &event.Session, &event.DateInWorld, &consequencesBytes)
		if err != nil {
			return nil, fmt.Errorf("scanning event: %w", err)
		}

		if len(consequencesBytes) > 0 {
			if err := json.Unmarshal(consequencesBytes, &event.Consequences); err != nil {
				return nil, fmt.Errorf("unmarshaling consequences: %w", err)
			}
		}

		event.Participants, err = c.fetchEventParticipants(ctx, entityID)
		if err != nil {
			return nil, err
		}
		event.Location, err = c.fetchEventLocations(ctx, entityID)
		if err != nil {
			return nil, err
		}

		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating events: %w", err)
	}

	if events == nil {
		events = []store.Event{}
	}

	return events, nil
}

func (c *Client) fetchEventParticipants(ctx context.Context, entityID int64) ([]string, error) {
	query := `
	SELECT p.name FROM edges ep
	JOIN entities p ON ep.dst_id = p.id
	WHERE ep.src_id = ? AND ep.rel_type = 'INVOLVES'
	`

	rows, err := c.db.QueryContext(ctx, query, entityID)
	if err != nil {
		return nil, fmt.Errorf("fetching participants: %w", err)
	}
	defer rows.Close()

	var participants []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scanning participant: %w", err)
		}
		participants = append(participants, name)
	}

	if participants == nil {
		participants = []string{}
	}
	return participants, nil
}

func (c *Client) fetchEventLocations(ctx context.Context, entityID int64) ([]string, error) {
	query := `
	SELECT l.name FROM edges el
	JOIN entities l ON el.dst_id = l.id
	WHERE el.src_id = ? AND el.rel_type = 'OCCURS_IN'
	`

	rows, err := c.db.QueryContext(ctx, query, entityID)
	if err != nil {
		return nil, fmt.Errorf("fetching locations: %w", err)
	}
	defer rows.Close()

	var locations []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scanning location: %w", err)
		}
		locations = append(locations, name)
	}

	if locations == nil {
		locations = []string{}
	}
	return locations, nil
}

func (c *Client) resolveCanonicalLayer(layerName string) (string, error) {
	byName := make(map[string]configLayer)
	for _, layer := range c.cfg.Layers {
		byName[strings.ToLower(layer.Name)] = configLayer{
			Name:      layer.Name,
			Canonical: layer.Canonical,
			DependsOn: layer.DependsOn,
		}
	}

	start, ok := byName[strings.ToLower(layerName)]
	if !ok {
		return "", fmt.Errorf("unknown layer: %s", layerName)
	}
	if start.Canonical {
		return start.Name, nil
	}

	var visit func(layer configLayer) (string, bool)
	visit = func(layer configLayer) (string, bool) {
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

type configLayer struct {
	Name      string
	Canonical bool
	DependsOn []string
}

func applyConsequences(props map[string]any, consequences []store.Consequence, target string) {
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
