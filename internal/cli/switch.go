package cli

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/idestis/pipe/internal/hub"
	"github.com/idestis/pipe/internal/resolve"
	"github.com/spf13/cobra"
)

var switchCreate string

func init() {
	switchCmd.Flags().StringVarP(&switchCreate, "create", "b", "", "create a new editable tag from the active tag's content")
}

var switchCmd = &cobra.Command{
	Use:   "switch <owner>/<name> [tag]",
	Short: "Switch the active tag for a hub pipeline",
	Args:  rangeArgs(1, 2, "pipe switch <owner>/<name> [tag]"),
	RunE: func(cmd *cobra.Command, args []string) error {
		owner, name, _ := resolve.ParsePipeArg(args[0])
		if owner == "" {
			return fmt.Errorf("owner required — use \"pipe switch <owner>/<name> [tag]\"")
		}
		log.Debug("switch target", "owner", owner, "name", name)

		idx, err := hub.LoadIndex(owner, name)
		if err != nil {
			return err
		}
		if idx == nil || len(idx.Tags) == 0 {
			return fmt.Errorf("no pulled tags for %s/%s — run \"pipe pull %s/%s\" first", owner, name, owner, name)
		}
		log.Debug("loaded index", "tags", len(idx.Tags), "activeTag", idx.ActiveTag)

		// --create / -b: create an editable tag from active tag's or blob's content
		if switchCreate != "" {
			log.Debug("create mode", "newTag", switchCreate)
			if err := validTag(switchCreate); err != nil {
				return fmt.Errorf("invalid tag %q: %w", switchCreate, err)
			}
			if _, ok := idx.Tags[switchCreate]; ok {
				return fmt.Errorf("tag %q already exists — use \"pipe switch %s/%s %s\" to switch to it", switchCreate, owner, name, switchCreate)
			}

			var content []byte
			sourceTag := idx.ActiveTag
			if sourceTag == "" {
				// No active tag — check if HEAD points to a blob
				headRef, herr := hub.ReadHeadRef(owner, name)
				if herr != nil || headRef.Kind != hub.HeadKindBlob {
					return fmt.Errorf("no active tag set for %s/%s", owner, name)
				}
				log.Debug("HEAD points to blob", "sha256", short(headRef.Value, 12))
				blobPath := hub.BlobPath(owner, name, headRef.Value)
				content, err = os.ReadFile(blobPath)
				if err != nil {
					return fmt.Errorf("reading blob %s: %w", short(headRef.Value, 12), err)
				}
				sourceTag = "sha256:" + short(headRef.Value, 12)
			} else {
				log.Debug("loading content from active tag", "sourceTag", sourceTag)
				content, err = hub.LoadContent(owner, name, sourceTag)
				if err != nil {
					return fmt.Errorf("reading active tag %q: %w", sourceTag, err)
				}
			}
			log.Debug("source content loaded", "from", sourceTag, "size", len(content))

			// Create as editable (regular file, independent copy)
			if err := hub.CreateEditableTag(owner, name, switchCreate, content); err != nil {
				return fmt.Errorf("creating editable tag: %w", err)
			}

			sha, md5h := hub.ComputeChecksums(content)
			log.Debug("editable tag checksums", "sha256", short(sha, 12), "md5", short(md5h, 12))
			idx.Tags[switchCreate] = hub.TagRecord{
				SHA256:    sha,
				MD5:       md5h,
				SizeBytes: int64(len(content)),
				CreatedAt: time.Now(),
				Editable:  true,
			}
			idx.ActiveTag = switchCreate
			log.Debug("setting HEAD", "tag", switchCreate)
			if err := hub.SetHead(owner, name, switchCreate); err != nil {
				return fmt.Errorf("setting HEAD: %w", err)
			}
			log.Debug("saving index", "activeTag", switchCreate)
			if err := hub.SaveIndex(idx); err != nil {
				return fmt.Errorf("saving index: %w", err)
			}

			log.Info("created editable tag", "pipe", owner+"/"+name, "tag", switchCreate, "from", sourceTag)
			log.Info("switched", "pipe", owner+"/"+name, "tag", switchCreate)
			return nil
		}

		var newTag string

		if len(args) >= 2 {
			// Explicit tag
			newTag = args[1]
			log.Debug("explicit tag requested", "tag", newTag)
			if _, ok := idx.Tags[newTag]; !ok {
				// Not a known tag — check if it looks like a SHA hex
				log.Debug("tag not in index, trying blob SHA match", "tag", newTag)
				matchedSHA, err := matchBlobSHA(owner, name, newTag)
				if err != nil {
					return fmt.Errorf("tag %q not pulled — available tags: %s", newTag, tagList(idx))
				}
				log.Debug("matched blob SHA", "input", newTag, "fullSHA", short(matchedSHA, 12))

				// Check if HEAD already points to this blob
				headRef, _ := hub.ReadHeadRef(owner, name)
				if headRef != nil && headRef.Kind == hub.HeadKindBlob && headRef.Value == matchedSHA {
					log.Debug("already on this blob", "sha256", short(matchedSHA, 12))
					fmt.Printf("%s/%s is already on blob sha256:%s\n", owner, name, short(matchedSHA, 12))
					return nil
				}

				log.Debug("setting HEAD to blob", "sha256", short(matchedSHA, 12))
				if err := hub.SetHeadBlob(owner, name, matchedSHA); err != nil {
					return fmt.Errorf("setting HEAD to blob: %w", err)
				}
				idx.ActiveTag = ""
				if err := hub.SaveIndex(idx); err != nil {
					return fmt.Errorf("saving index: %w", err)
				}

				log.Info("switched to blob", "pipe", owner+"/"+name, "sha256", short(matchedSHA, 12))
				return nil
			}
		} else {
			// Interactive selection — only show named tags
			tags := sortedTags(idx)
			fmt.Printf("Pulled tags for %s/%s:\n", owner, name)
			for i, t := range tags {
				active := ""
				if t == idx.ActiveTag {
					active = " (active)"
				}
				fmt.Printf("  [%d] %s%s\n", i+1, t, active)
			}

			// Show detached HEAD indicator if on a blob
			headRef, _ := hub.ReadHeadRef(owner, name)
			if headRef != nil && headRef.Kind == hub.HeadKindBlob {
				fmt.Printf("  * (detached) sha256:%s\n", short(headRef.Value, 12))
			}

			fmt.Print("\nSelect tag number: ")

			scanner := bufio.NewScanner(os.Stdin)
			scanner.Scan()
			input := scanner.Text()
			num, err := strconv.Atoi(input)
			if err != nil || num < 1 || num > len(tags) {
				return fmt.Errorf("invalid selection %q", input)
			}
			newTag = tags[num-1]
		}

		// Check if already active (tag mode)
		headRef, _ := hub.ReadHeadRef(owner, name)
		if headRef != nil && headRef.Kind == hub.HeadKindTag && headRef.Value == newTag {
			log.Debug("already on this tag", "tag", newTag)
			fmt.Printf("%s/%s is already on tag %q\n", owner, name, newTag)
			return nil
		}

		log.Debug("switching to tag", "from", idx.ActiveTag, "to", newTag)
		idx.ActiveTag = newTag
		if err := hub.SetHead(owner, name, newTag); err != nil {
			return fmt.Errorf("setting HEAD: %w", err)
		}
		log.Debug("saving index", "activeTag", newTag)
		if err := hub.SaveIndex(idx); err != nil {
			return fmt.Errorf("saving index: %w", err)
		}

		log.Info("switched", "pipe", owner+"/"+name, "tag", newTag)
		return nil
	},
}

func sortedTags(idx *hub.Index) []string {
	tags := make([]string, 0, len(idx.Tags))
	for t := range idx.Tags {
		tags = append(tags, t)
	}
	sort.Strings(tags)
	return tags
}

func tagList(idx *hub.Index) string {
	tags := sortedTags(idx)
	result := ""
	for i, t := range tags {
		if i > 0 {
			result += ", "
		}
		result += t
	}
	return result
}

// isHexString returns true if s is a non-empty string containing only hex characters.
func isHexString(s string) bool {
	if s == "" {
		return false
	}
	_, err := hex.DecodeString(strings.Repeat("0", len(s)%2) + s)
	if err != nil {
		return false
	}
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// matchBlobSHA finds a blob matching the given SHA (exact or prefix).
// Returns the full SHA hex or an error if no match or ambiguous.
func matchBlobSHA(owner, name, sha string) (string, error) {
	if !isHexString(sha) {
		return "", fmt.Errorf("not a valid hex string")
	}
	sha = strings.ToLower(sha)

	// Exact match (64-char SHA256)
	if len(sha) == 64 {
		blobPath := hub.BlobPath(owner, name, sha)
		if _, err := os.Stat(blobPath); err == nil {
			return sha, nil
		}
		return "", fmt.Errorf("blob %s not found", short(sha, 12))
	}

	// Prefix match
	blobDir := hub.BlobDir(owner, name)
	entries, err := os.ReadDir(blobDir)
	if err != nil {
		return "", fmt.Errorf("reading blob dir: %w", err)
	}

	var matches []string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), sha) {
			matches = append(matches, e.Name())
		}
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("no blob matching prefix %q", sha)
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("ambiguous prefix %q matches %d blobs", sha, len(matches))
	}
}
