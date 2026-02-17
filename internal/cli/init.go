package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/idestis/pipe/internal/config"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init <name>",
	Short: "Create a new pipeline",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		switch name {
		case "init", "list", "validate", "cache":
			return fmt.Errorf("%q is a reserved command name", name)
		}

		if !validName(name) {
			return fmt.Errorf("invalid pipeline name %q — use only letters, digits, hyphens, and underscores", name)
		}

		if err := os.MkdirAll(config.FilesDir, 0o755); err != nil {
			return fmt.Errorf("%s", friendlyError(err))
		}

		path := filepath.Join(config.FilesDir, name+".yaml")
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("pipeline %q already exists at %s", name, path)
		}

		template := fmt.Sprintf(`name: %s
description: ""
steps:
  - id: hello
    run: "echo Hello from %s"
`, name, name)

		if err := os.WriteFile(path, []byte(template), 0o644); err != nil {
			return fmt.Errorf("%s", friendlyError(err))
		}
		log.Info("created pipeline", "path", path)
		return nil
	},
}
