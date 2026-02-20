package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/deanmarano/dokkufile/pkg/apply"
	"github.com/deanmarano/dokkufile/pkg/plan"
	"github.com/deanmarano/dokkufile/pkg/schema"
	"github.com/deanmarano/dokkufile/pkg/state"
	"github.com/spf13/cobra"
)

func NewApplyCmd() *cobra.Command {
	var file string
	var dryRun bool
	var appFilter string
	var noCheckpoint bool
	var clearCheckpoint bool

	cmd := &cobra.Command{
		Use:   "apply [file]",
		Short: "Apply the Dokkufile to converge live state",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				file = args[0]
			}

			absFile, err := filepath.Abs(file)
			if err != nil {
				return fmt.Errorf("resolving path: %w", err)
			}
			checkpointDir := filepath.Dir(absFile)

			if clearCheckpoint {
				if err := apply.ClearCheckpointDir(checkpointDir); err != nil {
					return fmt.Errorf("clearing checkpoint: %w", err)
				}
				fmt.Println("Checkpoint cleared.")
				return nil
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
			fileRunner := &state.ExecFileRunner{}
			reader := &state.DokkuReader{Runner: runner, FileRunner: fileRunner}
			actual, err := reader.ReadScoped(desired)
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

			executor := &apply.Executor{
				Runner:     runner,
				FileRunner: fileRunner,
				DryRun:     dryRun,
			}
			if !noCheckpoint {
				executor.CheckpointDir = checkpointDir
				executor.DokkufilePath = absFile
			}
			return executor.Execute(p, desired, actual)
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "Dokkufile.yml", "path to Dokkufile")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print commands without executing them")
	cmd.Flags().StringVar(&appFilter, "app", "", "scope apply to a single app and its linked services")
	cmd.Flags().BoolVar(&noCheckpoint, "no-checkpoint", false, "disable checkpoint saving on failure")
	cmd.Flags().BoolVar(&clearCheckpoint, "clear-checkpoint", false, "delete any existing checkpoint and exit")
	return cmd
}
