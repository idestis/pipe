package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/charmbracelet/log"
	"github.com/getpipe-dev/pipe/internal/parser"
	"github.com/getpipe-dev/pipe/internal/resolve"
	"github.com/getpipe-dev/pipe/internal/runner"
	"github.com/spf13/cobra"
)

var lintCmd = &cobra.Command{
	Use:     "lint <name>",
	Aliases: []string{"validate"},
	Short:   "Lint a pipeline for errors and warnings",
	GroupID: "core",
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

		// Lint dot_file contents if configured.
		if pipeline.DotFile != "" {
			dotFileVars, dotFileWarns, dotErr := runner.ParseDotFile(pipeline.DotFile)
			switch {
			case errors.Is(dotErr, os.ErrNotExist):
				warns = append(warns, fmt.Sprintf("dot_file %q not found — use a full path or run from the directory containing the file", pipeline.DotFile))
			case dotErr != nil:
				warns = append(warns, fmt.Sprintf("dot_file %q: %v", pipeline.DotFile, dotErr))
			}
			warns = append(warns, dotFileWarns...)

			// Check for dot_file keys not declared in vars.
			_, resolveWarns := runner.ResolveVars(pipeline.Vars, dotFileVars, nil)
			warns = append(warns, resolveWarns...)
		}

		// Warn about PIPE_VAR_* env vars not matching declared vars.
		warns = append(warns, runner.UnmatchedEnvVarWarnings(pipeline.Vars)...)

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
