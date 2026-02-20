package cli

import (
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/getpipe-dev/pipe/internal/parser"
	"github.com/getpipe-dev/pipe/internal/resolve"
	"github.com/spf13/cobra"
)

var lintCmd = &cobra.Command{
	Use:     "lint <name>",
	Aliases: []string{"validate"},
	Short:   "Lint a pipeline for errors and warnings",
	Args:    exactArgs(1, "pipe lint <name>"),
	RunE: func(cmd *cobra.Command, args []string) error {
		ref, err := resolve.Resolve(args[0])
		if err != nil {
			return err
		}

		pipeline, err := parser.LoadPipelineFromPath(ref.Path, ref.Name)
		if err != nil {
			if isYAMLError(err) {
				return fmt.Errorf("invalid YAML in pipeline %q: %v", ref.Name, unwrapYAMLError(err))
			}
			return err
		}
		warns := parser.LintWarnings(pipeline)
		for _, w := range warns {
			log.Warn(w)
		}
		if len(warns) > 0 {
			fmt.Printf("pipeline %q is valid with %d warning(s)\n", ref.Name, len(warns))
		} else {
			fmt.Printf("pipeline %q is valid\n", ref.Name)
		}
		return nil
	},
}
