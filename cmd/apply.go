package cmd

import (
	"fmt"

	"github.com/deanmarano/dokkufile/pkg/apply"
	"github.com/deanmarano/dokkufile/pkg/plan"
	"github.com/deanmarano/dokkufile/pkg/schema"
	"github.com/deanmarano/dokkufile/pkg/state"
	"github.com/spf13/cobra"
)

func NewApplyCmd() *cobra.Command {
	var file string

	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Apply the Dokkufile to converge live state",
		RunE: func(cmd *cobra.Command, args []string) error {
			desired, err := schema.Load(file)
			if err != nil {
				return fmt.Errorf("loading dokkufile: %w", err)
			}

			runner := &state.ExecRunner{}
			reader := &state.DokkuReader{Runner: runner}
			actual, err := reader.Read()
			if err != nil {
				return fmt.Errorf("reading live state: %w", err)
			}

			p := plan.Diff(desired, actual)
			if len(p.Steps) == 0 {
				fmt.Println("No changes needed.")
				return nil
			}

			fmt.Print(p.String())
			fmt.Println()

			executor := &apply.Executor{Runner: runner}
			return executor.Execute(p, desired, actual)
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "Dokkufile.yml", "path to Dokkufile")
	return cmd
}
