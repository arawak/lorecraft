package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"lorecraft/internal/config"
	"lorecraft/internal/graph"
	"lorecraft/internal/validate"
)

func validateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Run consistency checks against the graph",
		RunE:  runValidate,
	}
	return cmd
}

func runValidate(cmd *cobra.Command, args []string) error {
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

	report, err := validate.Run(ctx, schema, client)
	if err != nil {
		return err
	}

	var errorIssues []validate.Issue
	var warnIssues []validate.Issue
	for _, issue := range report.Issues {
		switch issue.Severity {
		case validate.SeverityError:
			errorIssues = append(errorIssues, issue)
		case validate.SeverityWarn:
			warnIssues = append(warnIssues, issue)
		}
	}

	if len(errorIssues) == 0 && len(warnIssues) == 0 {
		fmt.Fprintln(os.Stdout, "No issues found.")
		return nil
	}

	if len(errorIssues) > 0 {
		fmt.Fprintf(os.Stdout, "Errors (%d):\n", len(errorIssues))
		printIssues(os.Stdout, errorIssues)
	}
	if len(warnIssues) > 0 {
		if len(errorIssues) > 0 {
			fmt.Fprintln(os.Stdout, "")
		}
		fmt.Fprintf(os.Stdout, "Warnings (%d):\n", len(warnIssues))
		printIssues(os.Stdout, warnIssues)
	}

	if len(errorIssues) > 0 {
		return fmt.Errorf("validation found errors")
	}
	return nil
}

func printIssues(out *os.File, issues []validate.Issue) {
	for _, issue := range issues {
		location := issue.Entity
		if issue.Layer != "" {
			location = fmt.Sprintf("%s [%s]", issue.Entity, issue.Layer)
		}
		if issue.FilePath != "" {
			location = fmt.Sprintf("%s (%s)", location, issue.FilePath)
		}
		fmt.Fprintf(out, "  - %s: %s (%s)\n", location, issue.Message, issue.Code)
	}
}
