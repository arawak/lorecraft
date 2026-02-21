package main

import "github.com/spf13/cobra"

func queryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query",
		Short: "Query the database from the CLI",
	}
	cmd.AddCommand(querySQLCmd())
	cmd.AddCommand(queryEntityCmd())
	cmd.AddCommand(queryRelationsCmd())
	cmd.AddCommand(queryListCmd())
	cmd.AddCommand(querySearchCmd())
	cmd.AddCommand(queryStateCmd())
	return cmd
}
