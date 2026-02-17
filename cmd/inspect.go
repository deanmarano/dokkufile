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
			actual, err := reader.Read()
			if err != nil {
				return fmt.Errorf("reading live state: %w", err)
			}

			// Filter to a single app unless --global
			if !global && len(args) > 0 {
				actual = filterByApp(actual, args[0])
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

// filterByApp returns a new Dokkufile containing only the specified app
// and its linked services.
func filterByApp(df *schema.Dokkufile, appName string) *schema.Dokkufile {
	app, ok := df.Apps[appName]
	if !ok {
		return &schema.Dokkufile{Version: df.Version}
	}

	filtered := &schema.Dokkufile{
		Version: df.Version,
		Apps:    map[string]schema.App{appName: app},
	}

	// Include linked services
	if len(app.Links) > 0 {
		filtered.Services = map[string]schema.Service{}
		for _, svcName := range app.Links {
			if svc, ok := df.Services[svcName]; ok {
				filtered.Services[svcName] = svc
			}
		}
	}

	// Include linked mail service
	if app.Mail != "" && df.MailServices != nil {
		if svc, ok := df.MailServices[app.Mail]; ok {
			filtered.MailServices = map[string]schema.MailService{app.Mail: svc}
		}
	}

	// Include linked auth directory
	if app.Auth != nil && app.Auth.Directory != "" && df.AuthDirectories != nil {
		if dir, ok := df.AuthDirectories[app.Auth.Directory]; ok {
			filtered.AuthDirectories = map[string]schema.AuthDirectory{app.Auth.Directory: dir}
		}
	}

	// Include protecting auth frontend
	if app.Auth != nil && app.Auth.Protected != "" && df.AuthFrontends != nil {
		if fe, ok := df.AuthFrontends[app.Auth.Protected]; ok {
			filtered.AuthFrontends = map[string]schema.AuthFrontend{app.Auth.Protected: fe}
		}
	}

	return filtered
}
