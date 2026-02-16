package cmd

import (
	"fmt"
	"os"

	"github.com/deanmarano/dokkufile/pkg/plan"
	"github.com/deanmarano/dokkufile/pkg/schema"
	"github.com/deanmarano/dokkufile/pkg/state"
	"github.com/spf13/cobra"
)

func NewPlanCmd() *cobra.Command {
	var file string

	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Compare desired state to live state and print a plan",
		RunE: func(cmd *cobra.Command, args []string) error {
			desired, err := schema.Load(file)
			if err != nil {
				return fmt.Errorf("loading dokkufile: %w", err)
			}

			reader := &state.DokkuReader{Runner: &state.ExecRunner{}}
			actual, err := reader.Read()
			if err != nil {
				return fmt.Errorf("reading live state: %w", err)
			}

			p := plan.Diff(desired, actual)
			fmt.Print(p.String())

			if len(p.Steps) > 0 {
				os.Exit(2) // non-zero to signal drift
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "Dokkufile.yml", "path to Dokkufile")
	return cmd
}
