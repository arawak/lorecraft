package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"lorecraft/internal/config"
	"lorecraft/internal/graph"
	"lorecraft/internal/ingest"
)

var ingestFull bool

func ingestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ingest",
		Short: "Synchronise the graph with markdown source files",
		RunE:  runIngest,
	}
	cmd.Flags().BoolVar(&ingestFull, "full", false, "Force full re-ingestion (ignore incremental hashes)")
	return cmd
}

func runIngest(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	cfg, err := config.LoadProjectConfig("lorecraft.yaml")
	if err != nil {
		return err
	}

	schema, err := config.LoadSchema("schema.yaml")
	if err != nil {
		return err
	}

	client, err := graph.NewClient(ctx, cfg.Neo4j.URI, cfg.Neo4j.Username, cfg.Neo4j.Password, cfg.Neo4j.Database)
	if err != nil {
		return err
	}
	defer client.Close(ctx)

	result, err := ingest.Run(ctx, cfg, schema, client, ingest.Options{Full: ingestFull})
	if err != nil {
		return err
	}

	fmt.Fprintln(os.Stdout, "Ingestion complete.")
	fmt.Fprintf(os.Stdout, "  Nodes upserted: %d\n", result.NodesUpserted)
	fmt.Fprintf(os.Stdout, "  Edges upserted: %d\n", result.EdgesUpserted)
	fmt.Fprintf(os.Stdout, "  Nodes removed:  %d\n", result.NodesRemoved)
	fmt.Fprintf(os.Stdout, "  Files skipped:  %d\n", result.FilesSkipped)

	if len(result.Errors) > 0 {
		fmt.Fprintf(os.Stdout, "\nErrors (%d):\n", len(result.Errors))
		for _, item := range result.Errors {
			fmt.Fprintf(os.Stdout, "  - %v\n", item)
		}
		return fmt.Errorf("ingestion completed with errors")
	}

	return nil
}
