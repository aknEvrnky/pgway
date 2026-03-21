package cmd

import (
	"fmt"
	"os"

	"github.com/aknEvrnky/pgway/internal/adapters/cli"
	"github.com/spf13/cobra"
)

func newApplyCmd(dispatcher *cli.Dispatcher) *cobra.Command {
	var filePath string

	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Apply resources from a YAML file",
		Example: `  pgctl apply -f proxy.yaml
  pgctl apply -f resources.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if filePath == "" {
				return fmt.Errorf("file path is required (-f)")
			}

			data, err := os.ReadFile(filePath)
			if err != nil {
				return fmt.Errorf("read file: %w", err)
			}

			resources, err := cli.ParseYAML(data)
			if err != nil {
				return fmt.Errorf("parse yaml: %w", err)
			}

			if len(resources) == 0 {
				fmt.Println("no resources found in file")
				return nil
			}

			return dispatcher.ApplyAll(cmd.Context(), resources)
		},
	}

	cmd.Flags().StringVarP(&filePath, "file", "f", "", "path to YAML resource file")

	return cmd
}
