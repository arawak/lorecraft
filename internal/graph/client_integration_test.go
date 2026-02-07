//go:build integration

package graph

import (
	"context"
	"testing"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"

	"lorecraft/internal/config"
)

func testClient(t *testing.T) *Client {
	t.Helper()
	ctx := context.Background()
	client, err := NewClient(ctx, "bolt://localhost:7687", "neo4j", "changeme", "neo4j")
	if err != nil {
		t.Fatalf("connecting to test neo4j: %v", err)
	}
	t.Cleanup(func() { _ = client.Close(ctx) })
	return client
}

func TestNewClient_Connect(t *testing.T) {
	_ = testClient(t)
}

func TestNewClient_BadCredentials(t *testing.T) {
	ctx := context.Background()
	client, err := NewClient(ctx, "bolt://localhost:7687", "neo4j", "wrong", "neo4j")
	if err == nil {
		_ = client.Close(ctx)
		t.Fatalf("expected error")
	}
}

func TestEnsureIndexes(t *testing.T) {
	ctx := context.Background()
	client := testClient(t)

	schema := &config.Schema{Version: 1}
	if err := client.EnsureIndexes(ctx, schema); err != nil {
		t.Fatalf("ensure indexes: %v", err)
	}
	if err := client.EnsureIndexes(ctx, schema); err != nil {
		t.Fatalf("ensure indexes (idempotent): %v", err)
	}

	session := client.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: client.database})
	defer session.Close(ctx)

	indexNames, err := listIndexNames(ctx, session)
	if err != nil {
		t.Fatalf("list indexes: %v", err)
	}

	requiredIndexes := []string{"entity_fulltext", "entity_layer", "entity_source_file"}
	for _, name := range requiredIndexes {
		if !contains(indexNames, name) {
			t.Fatalf("expected index %s", name)
		}
	}

	constraintNames, err := listConstraintNames(ctx, session)
	if err != nil {
		t.Fatalf("list constraints: %v", err)
	}
	if !contains(constraintNames, "entity_unique_name_layer") {
		t.Fatalf("expected constraint entity_unique_name_layer")
	}
}

func listIndexNames(ctx context.Context, session neo4j.SessionWithContext) ([]string, error) {
	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, "SHOW INDEXES YIELD name RETURN name", nil)
		if err != nil {
			return nil, err
		}
		var names []string
		for res.Next(ctx) {
			value, _ := res.Record().Get("name")
			if name, ok := value.(string); ok {
				names = append(names, name)
			}
		}
		if err := res.Err(); err != nil {
			return nil, err
		}
		return names, nil
	})
	if err != nil {
		return nil, err
	}
	return result.([]string), nil
}

func listConstraintNames(ctx context.Context, session neo4j.SessionWithContext) ([]string, error) {
	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, "SHOW CONSTRAINTS YIELD name RETURN name", nil)
		if err != nil {
			return nil, err
		}
		var names []string
		for res.Next(ctx) {
			value, _ := res.Record().Get("name")
			if name, ok := value.(string); ok {
				names = append(names, name)
			}
		}
		if err := res.Err(); err != nil {
			return nil, err
		}
		return names, nil
	})
	if err != nil {
		return nil, err
	}
	return result.([]string), nil
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
