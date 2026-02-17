package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/charmbracelet/log"
	"github.com/idestis/pipe/internal/parser"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate <name>",
	Short: "Validate a pipeline",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		pipeline, err := parser.LoadPipeline(name)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("pipeline %q not found\n  run \"pipe list\" to see available pipelines, or \"pipe init %s\" to create one", name, name)
			}
			if isYAMLError(err) {
				return fmt.Errorf("invalid YAML in pipeline %q: %v", name, unwrapYAMLError(err))
			}
			return err
		}
		for _, w := range parser.Warnings(pipeline) {
			log.Warn(w)
		}
		fmt.Printf("pipeline %q is valid\n", name)
		return nil
	},
}
