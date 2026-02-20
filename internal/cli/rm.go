package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/getpipe-dev/pipe/internal/config"
	"github.com/getpipe-dev/pipe/internal/hub"
	"github.com/getpipe-dev/pipe/internal/resolve"
	"github.com/spf13/cobra"
)

var rmCmd = &cobra.Command{
	Use:   "rm <name> or <owner>/<name>",
	Short:   "Remove a pipeline",
	GroupID: "core",
	Args:  exactArgs(1, "pipe rm <name>"),
	RunE: func(cmd *cobra.Command, args []string) error {
		owner, name, _ := resolve.ParsePipeArg(args[0])

		if owner != "" {
			return rmHubPipe(owner, name)
		}
		return rmLocalPipe(name)
	},
}

func rmLocalPipe(name string) error {
	path := filepath.Join(config.FilesDir, name+".yaml")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("local pipeline %q not found", name)
	}

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("removing pipeline: %w", err)
	}
	log.Info("removed local pipeline", "name", name)

	cleanupAliases(name)
	return nil
}

func rmHubPipe(owner, name string) error {
	dir := hub.PipePath(owner, name)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fmt.Errorf("hub pipeline %s/%s not found", owner, name)
	}

	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("removing pipeline: %w", err)
	}
	log.Info("removed hub pipeline", "pipe", owner+"/"+name)

	cleanupAliases(owner + "/" + name)
	return nil
}

func cleanupAliases(target string) {
	alias, err := resolve.FindAliasForTarget(target)
	if err != nil || alias == "" {
		return
	}
	if err := resolve.DeleteAlias(alias); err != nil {
		log.Warn("could not remove alias", "alias", alias, "err", err)
		return
	}
	log.Info("removed alias", "alias", alias)
}
