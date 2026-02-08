package main

import (
	"os"

	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "lorecraft",
		Short: "Graph-backed knowledge management system",
	}
	root.Version = version
	root.SetVersionTemplate("{{.Version}}\n")
	root.AddCommand(ingestCmd())
	root.AddCommand(serveCmd())
	root.AddCommand(validateCmd())
	root.AddCommand(queryCmd())
	root.AddCommand(initCmd())
	root.AddCommand(versionCmd())
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
