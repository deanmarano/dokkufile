package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/deanmarano/dokkufile/pkg/schema"
	"github.com/deanmarano/dokkufile/pkg/state"
	"github.com/spf13/cobra"
)

func NewDestroyCmd() *cobra.Command {
	var file string
	var appFilter string
	var force bool
	var includeServices bool

	cmd := &cobra.Command{
		Use:   "destroy [file]",
		Short: "Destroy apps (and optionally services) defined in the Dokkufile",
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

			// Show what will be destroyed
			if !force {
				fmt.Println("The following resources will be destroyed:")
				for name := range desired.Apps {
					fmt.Printf("  app: %s\n", name)
				}
				if includeServices {
					for name, svc := range desired.Services {
						fmt.Printf("  service (%s): %s\n", svc.Type, name)
					}
				}
				fmt.Print("\nAre you sure? (y/N): ")
				reader := bufio.NewReader(os.Stdin)
				answer, _ := reader.ReadString('\n')
				answer = strings.TrimSpace(strings.ToLower(answer))
				if answer != "y" && answer != "yes" {
					fmt.Println("Aborted.")
					return nil
				}
			}

			runner := &state.ExecRunner{}
			var errs []string

			// Destroy apps
			for name := range desired.Apps {
				fmt.Printf("-----> Destroying app %s\n", name)
				if _, err := runner.Run("apps:destroy", name, "--force"); err != nil {
					errs = append(errs, fmt.Sprintf("app %s: %v", name, err))
				}
			}

			// Destroy services if requested
			if includeServices {
				for name, svc := range desired.Services {
					fmt.Printf("-----> Destroying %s service %s\n", svc.Type, name)
					if _, err := runner.Run(svc.Type+":destroy", name, "--force"); err != nil {
						errs = append(errs, fmt.Sprintf("service %s: %v", name, err))
					}
				}
			}

			if len(errs) > 0 {
				return fmt.Errorf("errors during destroy:\n  %s", strings.Join(errs, "\n  "))
			}
			fmt.Println("=====> Destroy complete")
			return nil
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "Dokkufile.yml", "path to Dokkufile")
	cmd.Flags().StringVar(&appFilter, "app", "", "destroy only this app (and its linked services with --include-services)")
	cmd.Flags().BoolVar(&force, "force", false, "skip confirmation prompt")
	cmd.Flags().BoolVar(&includeServices, "include-services", false, "also destroy linked services")
	return cmd
}
