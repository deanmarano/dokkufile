package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/deanmarano/dokkufile/pkg/schema"
	"github.com/deanmarano/dokkufile/pkg/state"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func NewInspectCmd() *cobra.Command {
	var format string
	var global bool
	var includeEnv bool

	cmd := &cobra.Command{
		Use:   "inspect [app]",
		Short: "Dump live server state as a Dokkufile",
		Long:  "Inspect a single app (default) or all apps with --global. Env vars are omitted by default; use --include-env to include them.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && !global {
				return fmt.Errorf("specify an app name or use --global to inspect all apps")
			}

			reader := &state.DokkuReader{
				Runner:     &state.ExecRunner{},
				FileRunner: &state.ExecFileRunner{},
			}

			var actual *schema.Dokkufile
			if global {
				var err error
				actual, err = reader.Read()
				if err != nil {
					return fmt.Errorf("reading live state: %w", err)
				}
			} else {
				// Build a minimal scope with just the requested app name.
				scope := &schema.Dokkufile{
					Version: "1",
					Apps:    map[string]schema.App{args[0]: {}},
				}
				var err error
				actual, err = reader.ReadScoped(scope)
				if err != nil {
					return fmt.Errorf("reading live state: %w", err)
				}
				if len(actual.Apps) == 0 {
					return fmt.Errorf("app %q not found on server", args[0])
				}
			}

			// Strip env vars unless --include-env
			if !includeEnv {
				for name, app := range actual.Apps {
					app.Env = nil
					actual.Apps[name] = app
				}
			}

			switch format {
			case "json":
				data, err := json.MarshalIndent(actual, "", "  ")
				if err != nil {
					return err
				}
				fmt.Println(string(data))
			default:
				data, err := yaml.Marshal(actual)
				if err != nil {
					return err
				}
				fmt.Print(string(data))
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&format, "format", "yaml", "output format (yaml or json)")
	cmd.Flags().BoolVar(&global, "global", false, "inspect all apps on the server")
	cmd.Flags().BoolVar(&includeEnv, "include-env", false, "include environment variables (contains secrets)")
	return cmd
}
