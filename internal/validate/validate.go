package validate

import (
	"context"
	"fmt"
	"strings"

	"lorecraft/internal/config"
	"lorecraft/internal/graph"
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
	codeDuplicateName       = "duplicate_name"
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

func Run(ctx context.Context, schema *config.Schema, graphClient GraphValidator) (*Report, error) {
	if schema == nil {
		return nil, fmt.Errorf("schema is required")
	}
	if graphClient == nil {
		return nil, fmt.Errorf("graph client is required")
	}

	issues := make([]Issue, 0)

	entities, err := graphClient.ListEntities(ctx, "", "", "")
	if err != nil {
		return nil, fmt.Errorf("list entities: %w", err)
	}

	for _, summary := range entities {
		entity, err := graphClient.GetEntity(ctx, summary.Name, summary.EntityType)
		if err != nil {
			return nil, fmt.Errorf("get entity %s: %w", summary.Name, err)
		}
		if entity == nil {
			continue
		}
		entityType, ok := schema.EntityTypeByName(entity.EntityType)
		if !ok {
			continue
		}
		issues = append(issues, validateEnumValues(entity, entityType)...)
		issues = append(issues, validateRequiredProperties(entity, entityType)...)
	}

	placeholders, err := graphClient.ListDanglingPlaceholders(ctx)
	if err != nil {
		return nil, fmt.Errorf("list dangling placeholders: %w", err)
	}
	for _, summary := range placeholders {
		issues = append(issues, issueFromSummary(summary, SeverityError, codeDanglingPlaceholder, "dangling placeholder entity"))
	}

	orphans, err := graphClient.ListOrphanedEntities(ctx)
	if err != nil {
		return nil, fmt.Errorf("list orphaned entities: %w", err)
	}
	for _, summary := range orphans {
		issues = append(issues, issueFromSummary(summary, SeverityWarn, codeOrphanedEntity, "orphaned entity"))
	}

	duplicates, err := graphClient.ListDuplicateNames(ctx)
	if err != nil {
		return nil, fmt.Errorf("list duplicate names: %w", err)
	}
	for _, summary := range duplicates {
		issues = append(issues, issueFromSummary(summary, SeverityError, codeDuplicateName, "duplicate entity name in layer"))
	}

	crossLayer, err := graphClient.ListCrossLayerViolations(ctx)
	if err != nil {
		return nil, fmt.Errorf("list cross-layer violations: %w", err)
	}
	for _, summary := range crossLayer {
		issues = append(issues, issueFromSummary(summary, SeverityError, codeCrossLayerViolation, "cross-layer violation"))
	}

	return &Report{Issues: issues}, nil
}

func validateEnumValues(entity *graph.Entity, entityType *config.EntityType) []Issue {
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
		if !containsString(prop.Values, valueStr) {
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

func validateRequiredProperties(entity *graph.Entity, entityType *config.EntityType) []Issue {
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

func issueFromSummary(summary graph.EntitySummary, severity Severity, code, message string) Issue {
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
