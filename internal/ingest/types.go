package ingest

import "fmt"

type Consequence struct {
	Entity   string `json:"entity"`
	Property string `json:"property"`
	Value    any    `json:"value,omitempty"`
	Add      any    `json:"add,omitempty"`
}

func parseConsequences(value any) ([]Consequence, error) {
	if value == nil {
		return nil, nil
	}

	var items []any
	switch v := value.(type) {
	case []any:
		items = v
	case map[string]any:
		items = []any{v}
	default:
		return nil, fmt.Errorf("consequences must be a list")
	}

	consequences := make([]Consequence, 0, len(items))
	for i, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("consequence %d must be a map", i)
		}
		entity := toString(entry["entity"])
		property := toString(entry["property"])
		if entity == "" || property == "" {
			return nil, fmt.Errorf("consequence %d missing entity or property", i)
		}
		consequence := Consequence{Entity: entity, Property: property}
		if value, ok := entry["value"]; ok {
			consequence.Value = value
		}
		if add, ok := entry["add"]; ok {
			consequence.Add = add
		}
		consequences = append(consequences, consequence)
	}

	return consequences, nil
}

func toString(value any) string {
	if s, ok := value.(string); ok {
		return s
	}
	return ""
}
