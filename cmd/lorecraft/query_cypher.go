package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"lorecraft/internal/config"
	"lorecraft/internal/graph"
)

func queryCypherCmd() *cobra.Command {
	var paramPairs []string
	cmd := &cobra.Command{
		Use:   "cypher <query>",
		Short: "Execute a raw Cypher query",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.Join(args, " ")
			params, err := parseParams(paramPairs)
			if err != nil {
				return err
			}
			return runCypher(cmd, query, params)
		},
	}
	cmd.Flags().StringArrayVar(&paramPairs, "param", nil, "Query parameter as key=value (repeatable)")
	return cmd
}

func runCypher(cmd *cobra.Command, query string, params map[string]any) error {
	ctx := context.Background()

	cfg, err := config.LoadProjectConfig("lorecraft.yaml")
	if err != nil {
		return err
	}

	client, err := graph.NewClient(ctx, cfg.Neo4j.URI, cfg.Neo4j.Username, cfg.Neo4j.Password, cfg.Neo4j.Database)
	if err != nil {
		return err
	}
	defer client.Close(ctx)

	rows, err := client.RunCypher(ctx, query, params)
	if err != nil {
		return err
	}

	payload, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding result: %w", err)
	}
	fmt.Fprintln(os.Stdout, string(payload))
	return nil
}

func parseParams(pairs []string) (map[string]any, error) {
	params := make(map[string]any)
	for _, pair := range pairs {
		if pair == "" {
			continue
		}
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid param %q: expected key=value", pair)
		}
		key := strings.TrimSpace(parts[0])
		if key == "" {
			return nil, fmt.Errorf("invalid param %q: empty key", pair)
		}
		params[key] = strings.TrimSpace(parts[1])
	}
	return params, nil
}
