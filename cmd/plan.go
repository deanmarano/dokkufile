package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/deanmarano/dokkufile/pkg/plan"
	"github.com/deanmarano/dokkufile/pkg/schema"
	"github.com/deanmarano/dokkufile/pkg/state"
	"github.com/spf13/cobra"
)

func NewPlanCmd() *cobra.Command {
	var file string
	var format string
	var appFilter string

	cmd := &cobra.Command{
		Use:   "plan [file]",
		Short: "Compare desired state to live state and print a plan",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				file = args[0]
			}
			desired, err := schema.Load(file)
			if err != nil {
				return fmt.Errorf("loading dokkufile: %w", err)
			}

			if appFilter != "" {
				if _, ok := desired.Apps[appFilter]; !ok {
					return fmt.Errorf("app %q not found in dokkufile", appFilter)
				}
				desired = desired.FilterByApp(appFilter)
			}

			reader := &state.DokkuReader{
				Runner:     &state.ExecRunner{},
				FileRunner: &state.ExecFileRunner{},
			}
			actual, err := reader.ReadScoped(desired)
			if err != nil {
				return fmt.Errorf("reading live state: %w", err)
			}

			p := plan.Diff(desired, actual)

			switch format {
			case "json":
				data, err := json.MarshalIndent(p, "", "  ")
				if err != nil {
					return err
				}
				fmt.Println(string(data))
			default:
				fmt.Print(p.String())
			}

			if len(p.Steps) > 0 {
				os.Exit(2) // non-zero to signal drift
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "Dokkufile.yml", "path to Dokkufile")
	cmd.Flags().StringVar(&format, "format", "text", "output format (text or json)")
	cmd.Flags().StringVar(&appFilter, "app", "", "scope plan to a single app and its linked services")
	return cmd
}
