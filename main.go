package main

import (
	"fmt"
	"os"

	"github.com/deanmarano/dokkufile/cmd"
	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "dokkufile",
		Short: "Infrastructure as Code for Dokku",
		Long:  "Declarative configuration management for Dokku servers. Define desired state in a Dokkufile and let the CLI converge your server.",
	}

	root.AddCommand(cmd.NewPlanCmd())
	root.AddCommand(cmd.NewApplyCmd())
	root.AddCommand(cmd.NewInspectCmd())
	root.AddCommand(cmd.NewImportCmd())
	root.AddCommand(cmd.NewValidateCmd())
	root.AddCommand(cmd.NewVersionCmd())

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
