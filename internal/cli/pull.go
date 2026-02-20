package cli

import (
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/getpipe-dev/pipe/internal/auth"
	"github.com/getpipe-dev/pipe/internal/hub"
	"github.com/getpipe-dev/pipe/internal/parser"
	"github.com/getpipe-dev/pipe/internal/resolve"
	"github.com/spf13/cobra"
)

var pullForce bool

func init() {
	pullCmd.Flags().BoolVarP(&pullForce, "force", "f", false, "overwrite local changes")
}

var pullCmd = &cobra.Command{
	Use:   "pull <owner>/<name>[:<tag>]",
	Short:   "Pull a pipeline from Pipe Hub",
	GroupID: "hub",
	Args:  exactArgs(1, "pipe pull <owner>/<name>[:<tag>]"),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Auth is optional for pull — unauthenticated requests have lower rate limits.
		var client *hub.Client
		creds, err := auth.LoadCredentials()
		if err != nil {
			return fmt.Errorf("reading credentials: %w", err)
		}
		if creds != nil {
			log.Debug("using authenticated client")
			client = newHubClient(creds)
		} else {
			log.Debug("using unauthenticated client")
			client = newDefaultHubClient()
		}

		owner, name, tag := resolve.ParsePipeArg(args[0])
		if owner == "" {
			return fmt.Errorf("owner required — use \"pipe pull <owner>/<name>[:<tag>]\"")
		}
		if !validOwner(owner) {
			return fmt.Errorf("invalid owner name %q — must be 4-30 characters, using only lowercase letters, digits, hyphens, and dots", owner)
		}
		if tag == "" {
			tag = "latest"
		}
		log.Debug("pull target", "owner", owner, "name", name, "tag", tag)

		// Check for local modifications before overwriting
		if !pullForce {
			log.Debug("checking for local modifications", "owner", owner, "name", name, "tag", tag)
			dirty, err := hub.IsDirty(owner, name, tag)
			if err != nil {
				log.Warn("could not check for local changes", "err", err)
			} else if dirty {
				return fmt.Errorf("local changes to %s/%s:%s would be overwritten — push first or use --force", owner, name, tag)
			}
			log.Debug("dirty check result", "dirty", dirty)
		} else {
			log.Debug("skipping dirty check (--force)")
		}

		log.Info("fetching tag metadata", "pipe", owner+"/"+name, "tag", tag)
		detail, err := client.GetTag(owner, name, tag)
		if err != nil {
			return fmt.Errorf("fetching tag info: %w", err)
		}
		log.Debug("tag metadata", "sha256", short(detail.SHA256, 12), "md5", short(detail.MD5, 12), "size", detail.SizeBytes)

		log.Info("downloading content", "size", detail.SizeBytes)
		content, err := client.DownloadTag(owner, name, tag)
		if err != nil {
			return fmt.Errorf("downloading content: %w", err)
		}
		log.Debug("downloaded content", "size", len(content))

		// Verify checksum
		sha, _ := hub.ComputeChecksums(content)
		log.Debug("checksum verification", "local", short(sha, 12), "remote", short(detail.SHA256, 12), "match", sha == detail.SHA256)
		if sha != detail.SHA256 {
			return fmt.Errorf("checksum mismatch — expected %s, got %s", detail.SHA256, sha)
		}

		// Write content to disk
		log.Debug("saving content", "owner", owner, "name", name, "tag", tag)
		if err := hub.SaveContent(owner, name, tag, content); err != nil {
			return fmt.Errorf("saving content: %w", err)
		}

		// Update index
		log.Debug("updating index", "owner", owner, "name", name, "tag", tag, "sha256", short(sha, 12))
		if err := hub.UpdateIndex(owner, name, tag, detail.SHA256, detail.MD5, detail.SizeBytes); err != nil {
			return fmt.Errorf("updating index: %w", err)
		}

		// Validate YAML
		path := hub.ContentPath(owner, name, tag)
		log.Debug("validating YAML", "path", path)
		if _, err := parser.LoadPipelineFromPath(path, owner+"/"+name); err != nil {
			log.Warn("pulled content has validation issues", "err", err)
		}

		log.Info("pulled successfully", "pipe", owner+"/"+name, "tag", tag, "sha256", short(sha, 12))
		return nil
	},
}
