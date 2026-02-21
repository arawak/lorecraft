package validate

import (
	"context"
	"fmt"
	"strings"

	"lorecraft/internal/config"
	"lorecraft/internal/store"
)

type Severity string

const (
	SeverityError Severity = "error"
	SeverityWarn  Severity = "warning"
)

const (
	codeEnumInvalid         = "enum_value_invalid"
	codeMissingRequired     = "missing_required_property"
	codeDanglingPlaceholder = "dangling_placeholder"
	codeOrphanedEntity      = "orphaned_entity"
	codeCrossLayerViolation = "cross_layer_violation"
)

type Issue struct {
	Severity Severity
	Code     string
	Message  string
	Layer    string
	Entity   string
	FilePath string
}

type Report struct {
	Issues []Issue
}

func Run(ctx context.Context, schema *config.Schema, db Store) (*Report, error) {
	if schema == nil {
		return nil, fmt.Errorf("schema is required")
	}
	if db == nil {
		return nil, fmt.Errorf("database client is required")
	}

	issues := make([]Issue, 0)

	entities, err := db.ListEntitiesWithProperties(ctx)
	if err != nil {
		return nil, fmt.Errorf("list entities: %w", err)
	}

	for _, entity := range entities {
		entityType, ok := schema.EntityTypeByName(entity.EntityType)
		if !ok {
			continue
		}
		issues = append(issues, validateEnumValues(&entity, entityType)...)
		issues = append(issues, validateRequiredProperties(&entity, entityType)...)
	}

	placeholders, err := db.ListDanglingPlaceholders(ctx)
	if err != nil {
		return nil, fmt.Errorf("list dangling placeholders: %w", err)
	}
	for _, summary := range placeholders {
		issues = append(issues, issueFromSummary(summary, SeverityError, codeDanglingPlaceholder, "dangling placeholder entity"))
	}

	orphans, err := db.ListOrphanedEntities(ctx)
	if err != nil {
		return nil, fmt.Errorf("list orphaned entities: %w", err)
	}
	for _, summary := range orphans {
		issues = append(issues, issueFromSummary(summary, SeverityWarn, codeOrphanedEntity, "orphaned entity"))
	}

	crossLayer, err := db.ListCrossLayerViolations(ctx)
	if err != nil {
		return nil, fmt.Errorf("list cross-layer violations: %w", err)
	}
	for _, summary := range crossLayer {
		issues = append(issues, issueFromSummary(summary, SeverityError, codeCrossLayerViolation, "cross-layer violation"))
	}

	return &Report{Issues: issues}, nil
}

func validateEnumValues(entity *store.Entity, entityType *config.EntityType) []Issue {
	if entity == nil || entityType == nil {
		return nil
	}

	var issues []Issue
	for _, prop := range entityType.Properties {
		if !strings.EqualFold(prop.Type, "enum") || len(prop.Values) == 0 {
			continue
		}
		value, ok := entity.Properties[prop.Name]
		if !ok {
			continue
		}
		valueStr, ok := value.(string)
		if !ok {
			continue
		}
		if !containsStringCI(prop.Values, valueStr) {
			issues = append(issues, Issue{
				Severity: SeverityError,
				Code:     codeEnumInvalid,
				Message:  fmt.Sprintf("invalid enum value for %s: %s", prop.Name, valueStr),
				Layer:    entity.Layer,
				Entity:   entity.Name,
				FilePath: entity.SourceFile,
			})
		}
	}

	return issues
}

func validateRequiredProperties(entity *store.Entity, entityType *config.EntityType) []Issue {
	if entity == nil || entityType == nil {
		return nil
	}

	var issues []Issue
	for _, prop := range entityType.Properties {
		if !prop.Required {
			continue
		}
		value, ok := entity.Properties[prop.Name]
		if !ok || value == nil {
			issues = append(issues, Issue{
				Severity: SeverityError,
				Code:     codeMissingRequired,
				Message:  fmt.Sprintf("missing required property: %s", prop.Name),
				Layer:    entity.Layer,
				Entity:   entity.Name,
				FilePath: entity.SourceFile,
			})
			continue
		}
		if valueStr, ok := value.(string); ok && strings.TrimSpace(valueStr) == "" {
			issues = append(issues, Issue{
				Severity: SeverityError,
				Code:     codeMissingRequired,
				Message:  fmt.Sprintf("missing required property: %s", prop.Name),
				Layer:    entity.Layer,
				Entity:   entity.Name,
				FilePath: entity.SourceFile,
			})
		}
	}

	return issues
}

func issueFromSummary(summary store.EntitySummary, severity Severity, code, message string) Issue {
	return Issue{
		Severity: severity,
		Code:     code,
		Message:  message,
		Layer:    summary.Layer,
		Entity:   summary.Name,
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

// containsStringCI performs case-insensitive string matching against a list of enum values.
func containsStringCI(values []string, target string) bool {
	targetLower := strings.ToLower(target)
	for _, value := range values {
		if strings.ToLower(value) == targetLower {
			return true
		}
	}
	return false
}
