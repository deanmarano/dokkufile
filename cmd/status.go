package cmd

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/deanmarano/dokkufile/pkg/schema"
	"github.com/deanmarano/dokkufile/pkg/state"
	"github.com/spf13/cobra"
)

// appStatus holds parsed ps:report fields for an app.
type appStatus struct {
	Name     string `json:"name"`
	Running  string `json:"running"`
	Deployed string `json:"deployed"`
	Restore  string `json:"restore"`
}

// serviceStatus holds parsed service info for a service.
type serviceStatus struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Status string `json:"status"`
}

type statusOutput struct {
	Apps     []appStatus     `json:"apps"`
	Services []serviceStatus `json:"services"`
}

func NewStatusCmd() *cobra.Command {
	var file string
	var format string

	cmd := &cobra.Command{
		Use:   "status [file]",
		Short: "Show status of all apps and services in the Dokkufile",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				file = args[0]
			}
			desired, err := schema.Load(file)
			if err != nil {
				return fmt.Errorf("loading dokkufile: %w", err)
			}

			runner := &state.ExecRunner{}
			output := statusOutput{}

			// Collect app statuses
			appNames := sortedKeys(desired.Apps)
			for _, name := range appNames {
				as := appStatus{Name: name}
				if out, err := runner.Run("ps:report", name); err == nil {
					as.Running = parseReportField(out, "Ps running")
					as.Deployed = parseReportField(out, "Ps deployed")
					as.Restore = parseReportField(out, "Ps restore")
				} else {
					as.Running = "unknown"
					as.Deployed = "unknown"
					as.Restore = "unknown"
				}
				output.Apps = append(output.Apps, as)
			}

			// Collect service statuses
			svcNames := sortedKeys(desired.Services)
			for _, name := range svcNames {
				svc := desired.Services[name]
				ss := serviceStatus{Name: name, Type: svc.Type}
				if out, err := runner.Run(svc.Type+":info", name); err == nil {
					ss.Status = parseReportField(out, "Status")
					if ss.Status == "" {
						ss.Status = "exists"
					}
				} else {
					ss.Status = "not found"
				}
				output.Services = append(output.Services, ss)
			}

			if format == "json" {
				data, err := json.MarshalIndent(output, "", "  ")
				if err != nil {
					return err
				}
				fmt.Println(string(data))
				return nil
			}

			// Text output
			if len(output.Apps) > 0 {
				fmt.Println("=====> Apps")
				fmt.Printf("%-20s %-10s %-10s %-10s\n", "NAME", "RUNNING", "DEPLOYED", "RESTORE")
				for _, a := range output.Apps {
					fmt.Printf("%-20s %-10s %-10s %-10s\n", a.Name, a.Running, a.Deployed, a.Restore)
				}
			}

			if len(output.Services) > 0 {
				if len(output.Apps) > 0 {
					fmt.Println()
				}
				fmt.Println("=====> Services")
				fmt.Printf("%-20s %-15s %-10s\n", "NAME", "TYPE", "STATUS")
				for _, s := range output.Services {
					fmt.Printf("%-20s %-15s %-10s\n", s.Name, s.Type, s.Status)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "Dokkufile.yml", "path to Dokkufile")
	cmd.Flags().StringVar(&format, "format", "text", "output format (text or json)")
	return cmd
}

// parseReportField extracts a field value from dokku report output.
func parseReportField(output, fieldName string) string {
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		prefix := fieldName + ":"
		if strings.HasPrefix(trimmed, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, prefix))
		}
	}
	return ""
}

// sortedKeys returns the keys of a map sorted alphabetically.
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
