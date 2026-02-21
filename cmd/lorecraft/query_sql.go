package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"lorecraft/internal/config"
)

func querySQLCmd() *cobra.Command {
	var paramPairs []string
	cmd := &cobra.Command{
		Use:   "sql <query>",
		Short: "Execute a raw SQL query",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.Join(args, " ")
			params, err := parseParamPairs(paramPairs)
			if err != nil {
				return err
			}
			return runSQL(cmd, query, params)
		},
	}
	cmd.Flags().StringArrayVar(&paramPairs, "param", nil, "Query parameter as key=value (repeatable)")
	return cmd
}

func runSQL(cmd *cobra.Command, query string, params map[string]any) error {
	ctx := context.Background()

	cfg, err := config.LoadProjectConfig("lorecraft.yaml")
	if err != nil {
		return err
	}

	db, err := openDB(ctx, cfg)
	if err != nil {
		return err
	}
	defer db.Close(ctx)

	rows, err := db.RunSQL(ctx, query, params)
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

func parseParamPairs(pairs []string) (map[string]any, error) {
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
