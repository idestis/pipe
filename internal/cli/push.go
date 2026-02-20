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

var pushTags []string

func init() {
	pushCmd.Flags().StringArrayVarP(&pushTags, "tag", "t", nil, "tags to assign (repeatable, e.g. -t latest -t v2.0.0)")
}

var pushCmd = &cobra.Command{
	Use:   "push <owner>/<name>[:<tag>]",
	Short:   "Push a pipeline to Pipe Hub",
	GroupID: "hub",
	Args:  exactArgs(1, "pipe push <owner>/<name>[:<tag>]"),
	RunE: func(cmd *cobra.Command, args []string) error {
		creds, err := requireAuth()
		if err != nil {
			return err
		}

		owner, name, inlineTag := resolve.ParsePipeArg(args[0])
		if owner == "" {
			return fmt.Errorf("owner required — use \"pipe push <owner>/<name>[:<tag>]\"")
		}
		if !validOwner(owner) {
			return fmt.Errorf("invalid owner name %q — must be 4-30 characters, using only lowercase letters, digits, hyphens, and dots", owner)
		}

		// Build tag list: -t flags take precedence, then inline :tag, then default "latest"
		tags := pushTags
		if len(tags) == 0 {
			if inlineTag != "" {
				tags = []string{inlineTag}
			} else {
				tags = []string{"latest"}
			}
		}

		for _, t := range tags {
			if err := validTag(t); err != nil {
				return fmt.Errorf("invalid tag %q: %w", t, err)
			}
		}

		// Resolve source content: try exact tag file, then active tag, then local files
		var content []byte
		var sourceIsHub bool

		// Try the first requested tag's file directly
		hubPath := hub.ContentPath(owner, name, tags[0])
		log.Debug("resolving source content", "hubPath", hubPath)
		if _, err := os.Stat(hubPath); err == nil {
			content, err = os.ReadFile(hubPath)
			if err != nil {
				return fmt.Errorf("reading hub pipe: %w", err)
			}
			sourceIsHub = true
			log.Debug("resolved from tag file", "tag", tags[0], "size", len(content))
		} else {
			// Fall back to active tag
			idx, _ := hub.LoadIndex(owner, name)
			if idx != nil && idx.ActiveTag != "" {
				log.Debug("falling back to active tag", "activeTag", idx.ActiveTag)
				activeHubPath := hub.ContentPath(owner, name, idx.ActiveTag)
				if data, err := os.ReadFile(activeHubPath); err == nil {
					content = data
					sourceIsHub = true
					log.Debug("resolved from active tag", "tag", idx.ActiveTag, "size", len(content))
				}
			}
		}

		if content == nil {
			// Try local files
			localPath := filepath.Join(config.FilesDir, name+".yaml")
			log.Debug("trying local file", "path", localPath)
			data, err := os.ReadFile(localPath)
			if err != nil {
				return fmt.Errorf("pipe %q not found in hub store or local files", owner+"/"+name)
			}
			content = data
			log.Debug("resolved from local file", "size", len(content))
		}

		// Dirty check for editable tags
		if sourceIsHub {
			idx, _ := hub.LoadIndex(owner, name)
			if idx != nil {
				activeTag := idx.ActiveTag
				if activeTag != "" {
					rec, ok := idx.Tags[activeTag]
					if ok {
						dirty, derr := hub.IsDirty(owner, name, activeTag)
						if derr == nil && dirty {
							if rec.Editable {
								log.Warn("editable tag has local modifications, pushing current content", "tag", activeTag)
							} else {
								log.Warn("local modifications detected, pushing current content", "tag", activeTag)
							}
						}
					}
				}
			}
		}

		client := newHubClient(creds)

		// Check if pipe exists on hub
		log.Debug("checking if pipe exists on hub", "pipe", owner+"/"+name)
		meta, err := client.GetPipe(owner, name)
		if err != nil {
			return fmt.Errorf("checking pipe: %w", err)
		}
		if meta == nil {
			return fmt.Errorf("pipe %q not found on hub — create it first", owner+"/"+name)
		}
		log.Debug("pipe exists on hub", "pipe", owner+"/"+name)

		localSHA, localMD5 := hub.ComputeChecksums(content)
		log.Debug("local content checksums", "sha256", short(localSHA, 12), "md5", short(localMD5, 12), "size", len(content))

		// Pre-check: skip push if all tags already point to this content
		var newTags []string
		for _, t := range tags {
			remote, err := client.GetTag(owner, name, t)
			if err != nil {
				return fmt.Errorf("checking tag %q: %w", t, err)
			}
			if remote == nil {
				log.Debug("tag does not exist yet", "tag", t)
				newTags = append(newTags, t)
				continue
			}
			log.Debug("tag exists on hub", "tag", t, "remoteDigest", remote.Digest)
			if remote.Digest == "sha256:"+localSHA {
				log.Info("tag already up to date", "tag", t, "digest", short(remote.Digest, 19))
			} else if !meta.IsMutable && t != "latest" {
				return fmt.Errorf("tag %q already exists on immutable pipe %q with different content — cannot reassign",
					t, owner+"/"+name)
			} else {
				log.Debug("tag will be updated", "tag", t, "remote", short(remote.Digest, 19), "local", short(localSHA, 12))
				newTags = append(newTags, t)
			}
		}
		if len(newTags) == 0 {
			log.Info("all tags already up to date, nothing to push", "pipe", owner+"/"+name)
			return nil
		}
		tags = newTags

		log.Info("pushing", "pipe", owner+"/"+name, "tags", tags, "size", len(content))
		resp, err := client.Push(owner, name, content, tags)
		if err != nil {
			return fmt.Errorf("pushing: %w", err)
		}

		log.Debug("push response", "created", resp.Created, "digest", resp.Digest, "tags", resp.Tags, "size", resp.SizeBytes)

		// Verify response digest against local checksum
		expectedDigest := "sha256:" + localSHA
		if resp.Digest != expectedDigest {
			return fmt.Errorf("digest mismatch after push — local %s, remote %s", expectedDigest, resp.Digest)
		}

		// Re-snapshot: write pushed content as a correctly-named blob,
		// re-point the active tag symlink, and update its index record.
		if sourceIsHub {
			idx, _ := hub.LoadIndex(owner, name)
			if idx != nil && idx.ActiveTag != "" {
				newSha, err := hub.WriteBlob(owner, name, content)
				if err != nil {
					return fmt.Errorf("writing blob after push: %w", err)
				}
				if err := hub.CreateTagSymlink(owner, name, idx.ActiveTag, newSha); err != nil {
					return fmt.Errorf("re-pointing active tag: %w", err)
				}
				idx.Tags[idx.ActiveTag] = hub.TagRecord{
					SHA256:    localSHA,
					MD5:       localMD5,
					SizeBytes: resp.SizeBytes,
					PulledAt:  idx.Tags[idx.ActiveTag].PulledAt,
				}
				if err := hub.SaveIndex(idx); err != nil {
					return fmt.Errorf("updating index: %w", err)
				}
				if err := hub.GarbageCollectBlobs(owner, name); err != nil {
					log.Warn("garbage collection failed", "err", err)
				}
			}
		}

		if resp.Created {
			log.Info("pushed successfully", "pipe", owner+"/"+name, "tags", resp.Tags, "digest", short(resp.Digest, 19))
		} else {
			log.Info("content already exists", "pipe", owner+"/"+name, "tags", resp.Tags, "digest", short(resp.Digest, 19))
		}
		return nil
	},
}
