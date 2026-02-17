package cmd

import (
	"fmt"

	"github.com/deanmarano/dokkufile/pkg/schema"
	"github.com/spf13/cobra"
)

func NewValidateCmd() *cobra.Command {
	var file string

	cmd := &cobra.Command{
		Use:   "validate [file]",
		Short: "Validate a Dokkufile without connecting to the server",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				file = args[0]
			}
			_, err := schema.Load(file)
			if err != nil {
				return fmt.Errorf("validation failed: %w", err)
			}

			fmt.Println("Dokkufile is valid.")
			return nil
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "Dokkufile.yml", "path to Dokkufile")
	return cmd
}
