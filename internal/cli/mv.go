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

var mvCmd = &cobra.Command{
	Use:   "mv <name> <owner>/<name>",
	Short: "Convert a local pipeline to a hub namespace",
	Long:  "Converts a local pipeline from ~/.pipe/files/ into the hub layout under ~/.pipe/hub/, enabling tagging, versioning, and pushing to Pipe Hub.",
	Args:  exactArgs(2, "pipe mv <name> <owner>/<name>"),
	RunE: func(cmd *cobra.Command, args []string) error {
		srcName := args[0]
		dstArg := args[1]

		dstOwner, dstName, _ := resolve.ParsePipeArg(dstArg)
		if dstOwner == "" {
			return fmt.Errorf("destination must include owner — use \"pipe mv %s <owner>/%s\"", srcName, srcName)
		}
		if !validOwner(dstOwner) {
			return fmt.Errorf("invalid owner name %q — must be 4-30 characters, using only lowercase letters, digits, hyphens, and dots", dstOwner)
		}

		// Verify source exists
		srcPath := filepath.Join(config.FilesDir, srcName+".yaml")
		content, err := os.ReadFile(srcPath)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("local pipeline %q not found", srcName)
			}
			return fmt.Errorf("reading pipeline: %w", err)
		}

		idx, _ := hub.LoadIndex(dstOwner, dstName)

		if idx != nil {
			// Existing hub pipe — import as untagged blob referenced by SHA
			sha, err := hub.WriteBlob(dstOwner, dstName, content)
			if err != nil {
				return fmt.Errorf("writing blob: %w", err)
			}

			if err := hub.SetHeadBlob(dstOwner, dstName, sha); err != nil {
				return fmt.Errorf("setting HEAD to blob: %w", err)
			}

			idx.ActiveTag = ""
			if err := hub.SaveIndex(idx); err != nil {
				return fmt.Errorf("saving index: %w", err)
			}

			// Delete original
			if err := os.Remove(srcPath); err != nil {
				log.Warn("could not remove original file", "path", srcPath, "err", err)
			}

			log.Info("pipe already exists, imported as untagged blob", "pipe", dstOwner+"/"+dstName)
			log.Info("HEAD", "ref", "sha256:"+short(sha, 12))
		} else {
			// New hub pipe — create with "latest" tag
			tag := "latest"
			if err := hub.SaveContent(dstOwner, dstName, tag, content); err != nil {
				return fmt.Errorf("saving to hub: %w", err)
			}

			sha, md5sum := hub.ComputeChecksums(content)
			if err := hub.UpdateIndex(dstOwner, dstName, tag, sha, md5sum, int64(len(content))); err != nil {
				return fmt.Errorf("creating index: %w", err)
			}

			// Delete original
			if err := os.Remove(srcPath); err != nil {
				log.Warn("could not remove original file", "path", srcPath, "err", err)
			}

			log.Info("moved pipeline", "from", srcName, "to", dstOwner+"/"+dstName)
		}

		// Auto-create alias so "pipe run <name>" keeps working
		if err := resolve.SetAlias(srcName, dstOwner+"/"+dstName); err != nil {
			return fmt.Errorf("creating alias: %w", err)
		}
		log.Info("created alias", "alias", srcName, "target", dstOwner+"/"+dstName)

		return nil
	},
}
