package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/idestis/pipe/internal/config"
	"github.com/idestis/pipe/internal/hub"
	"github.com/idestis/pipe/internal/resolve"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init <name>",
	Short: "Create a new pipeline",
	Args:  exactArgs(1, "pipe init <name>"),
	RunE: func(cmd *cobra.Command, args []string) error {
		arg := args[0]

		owner, name, _ := resolve.ParsePipeArg(arg)

		// Validate the name portion
		if !validName(name) {
			return fmt.Errorf("invalid pipeline name %q — use only letters, digits, hyphens, and underscores", name)
		}
		if owner != "" && !validName(owner) {
			return fmt.Errorf("invalid owner name %q — use only letters, digits, hyphens, and underscores", owner)
		}

		if reservedNames[name] {
			return fmt.Errorf("%q is a reserved command name", name)
		}

		if owner != "" {
			// Hub pipe: scaffold in ~/.pipe/hub/{owner}/{name}/
			return initHubPipe(owner, name)
		}

		// Local pipe: scaffold in ~/.pipe/files/
		return initLocalPipe(name)
	},
}

func initLocalPipe(name string) error {
	log.Debug("creating local pipe directory", "dir", config.FilesDir)
	if err := os.MkdirAll(config.FilesDir, 0o755); err != nil {
		return fmt.Errorf("%s", friendlyError(err))
	}

	path := filepath.Join(config.FilesDir, name+".yaml")
	log.Debug("checking for existing pipeline", "path", path)
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("pipeline %q already exists at %s", name, path)
	}

	template := fmt.Sprintf(`name: %s
description: ""
# vars:
#   GREETING: "Hello"
steps:
  - id: hello
    run: "echo Hello from %s"
`, name, name)

	if err := os.WriteFile(path, []byte(template), 0o644); err != nil {
		return fmt.Errorf("%s", friendlyError(err))
	}
	log.Info("created pipeline", "path", path)
	return nil
}

func initHubPipe(owner, name string) error {
	log.Debug("checking for existing hub index", "owner", owner, "name", name)
	idx, _ := hub.LoadIndex(owner, name)
	if idx != nil {
		return fmt.Errorf("hub pipe %s/%s already exists", owner, name)
	}

	tag := "latest"
	displayName := owner + "/" + name
	template := fmt.Sprintf(`name: %s
description: ""
# vars:
#   GREETING: "Hello"
steps:
  - id: hello
    run: "echo Hello from %s"
`, displayName, displayName)

	content := []byte(template)
	log.Debug("saving hub content", "owner", owner, "name", name, "tag", tag, "size", len(content))
	if err := hub.SaveContent(owner, name, tag, content); err != nil {
		return fmt.Errorf("saving content: %w", err)
	}

	sha, md5sum := hub.ComputeChecksums(content)
	log.Debug("content checksums", "sha256", short(sha, 12), "md5", short(md5sum, 12))
	if err := hub.UpdateIndex(owner, name, tag, sha, md5sum, int64(len(content))); err != nil {
		return fmt.Errorf("creating index: %w", err)
	}
	log.Debug("index created", "owner", owner, "name", name, "tag", tag)

	path := hub.ContentPath(owner, name, tag)
	log.Info("created hub pipeline", "path", path)

	// Create alias if name doesn't conflict with a local pipeline or reserved word
	simpleName := strings.ToLower(name)
	localPath := filepath.Join(config.FilesDir, name+".yaml")
	log.Debug("checking alias eligibility", "simpleName", simpleName, "localPath", localPath)
	if reservedNames[simpleName] {
		log.Debug("skipping alias, reserved name", "name", simpleName)
	} else if _, err := os.Stat(localPath); err == nil {
		log.Warn("skipping alias — local pipeline with the same name exists", "name", name)
	} else {
		_ = resolve.SetAlias(name, displayName)
		log.Info("created alias", "alias", name, "target", displayName)
	}

	return nil
}
