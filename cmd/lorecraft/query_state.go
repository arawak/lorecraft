package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"lorecraft/internal/config"
)

func queryStateCmd() *cobra.Command {
	var layer string
	cmd := &cobra.Command{
		Use:   "state <name>",
		Short: "Compute current state from campaign events",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(layer) == "" {
				return fmt.Errorf("--layer is required")
			}
			name := args[0]
			return runQueryState(cmd, name, layer)
		},
	}
	cmd.Flags().StringVar(&layer, "layer", "", "Campaign layer to evaluate")
	return cmd
}

func runQueryState(cmd *cobra.Command, name, layer string) error {
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

	state, err := db.GetCurrentState(ctx, name, layer)
	if err != nil {
		return err
	}
	if state == nil {
		fmt.Fprintf(os.Stdout, "No state found for %q in layer %q.\n", name, layer)
		return nil
	}

	printPropertyBlock("Base properties", state.BaseProperties)

	if len(state.Events) > 0 {
		fmt.Fprintln(os.Stdout, "Events:")
		for _, event := range state.Events {
			fmt.Fprintf(os.Stdout, "  [%d] %s (%s)\n", event.Session, event.Name, event.Layer)
			if event.DateInWorld != "" {
				fmt.Fprintf(os.Stdout, "    Date: %s\n", event.DateInWorld)
			}
			if len(event.Participants) > 0 {
				fmt.Fprintf(os.Stdout, "    Participants: %s\n", joinValues(event.Participants))
			}
			if len(event.Location) > 0 {
				fmt.Fprintf(os.Stdout, "    Location: %s\n", joinValues(event.Location))
			}
			if len(event.Consequences) > 0 {
				fmt.Fprintln(os.Stdout, "    Consequences:")
				for _, consequence := range event.Consequences {
					if consequence.Value != nil {
						fmt.Fprintf(os.Stdout, "      - %s.%s = %v\n", consequence.Entity, consequence.Property, consequence.Value)
						continue
					}
					if consequence.Add != nil {
						fmt.Fprintf(os.Stdout, "      - %s.%s += %v\n", consequence.Entity, consequence.Property, consequence.Add)
					}
				}
			}
		}
		fmt.Fprintln(os.Stdout, "")
	}

	printPropertyBlock("Current properties", state.CurrentProperties)
	return nil
}

func printPropertyBlock(title string, props map[string]any) {
	if len(props) == 0 {
		return
	}
	keys := make([]string, 0, len(props))
	for key := range props {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	fmt.Fprintf(os.Stdout, "%s:\n", title)
	for _, key := range keys {
		fmt.Fprintf(os.Stdout, "  %s: %v\n", key, props[key])
	}
	fmt.Fprintln(os.Stdout, "")
}
