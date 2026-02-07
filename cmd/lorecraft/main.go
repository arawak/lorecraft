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
	root.AddCommand(ingestCmd())
	root.AddCommand(serveCmd())
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
