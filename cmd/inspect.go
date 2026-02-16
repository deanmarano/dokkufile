package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/deanmarano/dokkufile/pkg/state"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func NewInspectCmd() *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:   "inspect",
		Short: "Dump live server state as a Dokkufile",
		RunE: func(cmd *cobra.Command, args []string) error {
			reader := &state.DokkuReader{Runner: &state.ExecRunner{}}
			actual, err := reader.Read()
			if err != nil {
				return fmt.Errorf("reading live state: %w", err)
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
	return cmd
}
