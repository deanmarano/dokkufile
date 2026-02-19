package cmd

import (
	"fmt"
	"strings"

	"github.com/deanmarano/dokkufile/pkg/schema"
	"github.com/deanmarano/dokkufile/pkg/state"
	"github.com/spf13/cobra"
)

func NewRestartCmd() *cobra.Command {
	var file string
	var appFilter string

	cmd := &cobra.Command{
		Use:   "restart [file]",
		Short: "Restart all apps defined in the Dokkufile",
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

			runner := &state.ExecRunner{}
			var errs []string
			for name := range desired.Apps {
				fmt.Printf("-----> Restarting %s\n", name)
				if _, err := runner.Run("ps:restart", name); err != nil {
					errs = append(errs, fmt.Sprintf("%s: %v", name, err))
				}
			}

			if len(errs) > 0 {
				return fmt.Errorf("errors restarting apps:\n  %s", strings.Join(errs, "\n  "))
			}
			fmt.Println("=====> All apps restarted")
			return nil
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "Dokkufile.yml", "path to Dokkufile")
	cmd.Flags().StringVar(&appFilter, "app", "", "restart only this app")
	return cmd
}
