package cmd

import (
	"fmt"
	"os"

	"github.com/deanmarano/dokkufile/pkg/compose"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func NewImportCmd() *cobra.Command {
	var file string

	cmd := &cobra.Command{
		Use:   "import",
		Short: "Generate a Dokkufile from a docker-compose file",
		RunE: func(cmd *cobra.Command, args []string) error {
			data, err := os.ReadFile(file)
			if err != nil {
				return fmt.Errorf("reading compose file: %w", err)
			}

			df, err := compose.ImportCompose(data)
			if err != nil {
				return err
			}

			out, err := yaml.Marshal(df)
			if err != nil {
				return err
			}

			fmt.Print(string(out))
			return nil
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "docker-compose.yml", "path to docker-compose file")
	return cmd
}
