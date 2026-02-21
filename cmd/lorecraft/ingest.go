package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"lorecraft/internal/config"
	"lorecraft/internal/ingest"
)

var ingestFull bool

func ingestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ingest",
		Short: "Synchronise the database with markdown source files",
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

	db, err := openDB(ctx, cfg)
	if err != nil {
		return err
	}
	defer db.Close(ctx)

	result, err := ingest.Run(ctx, cfg, schema, db, ingest.Options{Full: ingestFull})
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
