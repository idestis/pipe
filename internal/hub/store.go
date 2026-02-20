package hub

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/getpipe-dev/pipe/internal/config"
)

// PipePath returns the directory for a hub pipe: ~/.pipe/hub/{owner}/{name}
func PipePath(owner, name string) string {
	return filepath.Join(config.HubDir, owner, name)
}

// BlobDir returns the blob storage directory for a hub pipe.
func BlobDir(owner, name string) string {
	return filepath.Join(PipePath(owner, name), "blobs", "sha256")
}

// BlobPath returns the path to a specific blob by sha256 hex.
func BlobPath(owner, name, sha256Hex string) string {
	return filepath.Join(BlobDir(owner, name), sha256Hex)
}

// TagDir returns the tags directory for a hub pipe.
func TagDir(owner, name string) string {
	return filepath.Join(PipePath(owner, name), "tags")
}

// TagPath returns the path to a specific tag (symlink or regular file).
func TagPath(owner, name, tag string) string {
	return filepath.Join(TagDir(owner, name), tag)
}

// HeadPath returns the path to the HEAD symlink.
func HeadPath(owner, name string) string {
	return filepath.Join(PipePath(owner, name), "HEAD")
}

// ContentPath returns the tag path for a specific tag.
// os.ReadFile follows symlinks, so this works for both symlink and editable tags.
func ContentPath(owner, name, tag string) string {
	return TagPath(owner, name, tag)
}

// IndexPath returns the index.json path for a hub pipe.
func IndexPath(owner, name string) string {
	return filepath.Join(PipePath(owner, name), "index.json")
}

// WriteBlob writes content to the blob store (content-addressable).
// Returns the sha256 hex digest. Skips writing if the blob already exists.
// Uses atomic write via tmp+rename.
func WriteBlob(owner, name string, content []byte) (string, error) {
	sha, _ := ComputeChecksums(content)
	dir := BlobDir(owner, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creating blob dir: %w", err)
	}
	blobPath := BlobPath(owner, name, sha)
	if _, err := os.Stat(blobPath); err == nil {
		return sha, nil // already exists
	}
	tmp := blobPath + ".tmp"
	if err := os.WriteFile(tmp, content, 0o644); err != nil {
		return "", fmt.Errorf("writing blob: %w", err)
	}
	if err := os.Rename(tmp, blobPath); err != nil {
		return "", fmt.Errorf("renaming blob: %w", err)
	}
	return sha, nil
}

// CreateTagSymlink creates or replaces a tag as a relative symlink pointing
// to ../../blobs/sha256/{hex}.
func CreateTagSymlink(owner, name, tag, sha256Hex string) error {
	dir := TagDir(owner, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating tags dir: %w", err)
	}
	tagPath := TagPath(owner, name, tag)
	// Remove existing tag (symlink or file)
	_ = os.Remove(tagPath)
	target := filepath.Join("..", "blobs", "sha256", sha256Hex)
	return os.Symlink(target, tagPath)
}

// CreateEditableTag writes a tag as a regular file (independent copy) for editing.
func CreateEditableTag(owner, name, tag string, content []byte) error {
	dir := TagDir(owner, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating tags dir: %w", err)
	}
	tagPath := TagPath(owner, name, tag)
	_ = os.Remove(tagPath) // remove any existing symlink or file
	return os.WriteFile(tagPath, content, 0o644)
}

// IsTagEditable checks whether a tag is a regular file (editable) or symlink.
func IsTagEditable(owner, name, tag string) (bool, error) {
	tagPath := TagPath(owner, name, tag)
	fi, err := os.Lstat(tagPath)
	if err != nil {
		return false, err
	}
	return fi.Mode().IsRegular(), nil
}

// SetHead creates or replaces the HEAD symlink to point to tags/{tag}.
func SetHead(owner, name, tag string) error {
	headPath := HeadPath(owner, name)
	_ = os.Remove(headPath)
	target := filepath.Join("tags", tag)
	return os.Symlink(target, headPath)
}

// SetHeadBlob creates or replaces the HEAD symlink to point to blobs/sha256/{hex}.
func SetHeadBlob(owner, name, sha256Hex string) error {
	headPath := HeadPath(owner, name)
	_ = os.Remove(headPath)
	target := filepath.Join("blobs", "sha256", sha256Hex)
	return os.Symlink(target, headPath)
}

// ReadHeadRef reads the HEAD symlink and returns a typed HeadRef.
// Falls back to idx.ActiveTag as HeadKindTag if HEAD symlink is missing.
func ReadHeadRef(owner, name string) (*HeadRef, error) {
	headPath := HeadPath(owner, name)
	target, err := os.Readlink(headPath)
	if err == nil {
		if strings.HasPrefix(target, filepath.Join("blobs", "sha256")+string(filepath.Separator)) {
			return &HeadRef{Kind: HeadKindBlob, Value: filepath.Base(target)}, nil
		}
		return &HeadRef{Kind: HeadKindTag, Value: filepath.Base(target)}, nil
	}
	// Fall back to index
	idx, err := loadIndexRaw(owner, name)
	if err != nil {
		return nil, err
	}
	if idx == nil {
		return nil, fmt.Errorf("no HEAD or index for %s/%s", owner, name)
	}
	return &HeadRef{Kind: HeadKindTag, Value: idx.ActiveTag}, nil
}

// ReadHead reads the HEAD symlink target and returns the active tag name.
// Falls back to loading ActiveTag from the index if HEAD doesn't exist.
func ReadHead(owner, name string) (string, error) {
	ref, err := ReadHeadRef(owner, name)
	if err != nil {
		return "", err
	}
	return ref.Value, nil
}

// LoadIndex reads and parses the index.json for a hub pipe.
// If the schema version is < 2, it auto-migrates to the new layout.
// Returns nil and no error if the file does not exist.
func LoadIndex(owner, name string) (*Index, error) {
	idx, err := loadIndexRaw(owner, name)
	if err != nil || idx == nil {
		return idx, err
	}
	if idx.SchemaVersion < 2 {
		if err := MigrateV1ToV2(owner, name); err != nil {
			return nil, fmt.Errorf("migrating to v2: %w", err)
		}
		// Re-read after migration
		return loadIndexRaw(owner, name)
	}
	return idx, nil
}

// loadIndexRaw reads index.json without triggering migration.
func loadIndexRaw(owner, name string) (*Index, error) {
	path := IndexPath(owner, name)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading index: %w", err)
	}
	var idx Index
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("parsing index: %w", err)
	}
	return &idx, nil
}

// SaveIndex writes the index.json for a hub pipe atomically.
func SaveIndex(idx *Index) error {
	dir := PipePath(idx.Owner, idx.Name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}
	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling index: %w", err)
	}
	path := IndexPath(idx.Owner, idx.Name)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("writing index: %w", err)
	}
	return os.Rename(tmp, path)
}

// SaveContent writes content to the blob store and creates a tag symlink.
func SaveContent(owner, name, tag string, content []byte) error {
	sha, err := WriteBlob(owner, name, content)
	if err != nil {
		return err
	}
	return CreateTagSymlink(owner, name, tag, sha)
}

// LoadContent reads the content for a tag from disk.
// os.ReadFile follows symlinks, so this works for both symlink and editable tags.
func LoadContent(owner, name, tag string) ([]byte, error) {
	path := TagPath(owner, name, tag)
	return os.ReadFile(path)
}

// ComputeChecksums returns sha256 and md5 hex digests for the given data.
func ComputeChecksums(data []byte) (sha256Hex, md5Hex string) {
	s := sha256.Sum256(data)
	m := md5.Sum(data)
	return hex.EncodeToString(s[:]), hex.EncodeToString(m[:])
}

// IsDirty returns true if the on-disk content for a tag differs from its index checksum.
// Returns false if the tag has no index record or the file doesn't exist.
func IsDirty(owner, name, tag string) (bool, error) {
	idx, err := LoadIndex(owner, name)
	if err != nil {
		return false, err
	}
	if idx == nil {
		return false, nil
	}
	rec, ok := idx.Tags[tag]
	if !ok {
		return false, nil
	}
	content, err := LoadContent(owner, name, tag)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	sha, _ := ComputeChecksums(content)
	return sha != rec.SHA256, nil
}

// VerifyChecksum compares the sha256 of content on disk against the index record.
func VerifyChecksum(owner, name, tag string) error {
	idx, err := LoadIndex(owner, name)
	if err != nil {
		return err
	}
	if idx == nil {
		return fmt.Errorf("no index found for %s/%s", owner, name)
	}
	rec, ok := idx.Tags[tag]
	if !ok {
		return fmt.Errorf("tag %q not found in index for %s/%s", tag, owner, name)
	}
	content, err := LoadContent(owner, name, tag)
	if err != nil {
		return fmt.Errorf("reading content: %w", err)
	}
	sha, _ := ComputeChecksums(content)
	if sha != rec.SHA256 {
		return fmt.Errorf("checksum mismatch for %s/%s:%s — expected %s, got %s", owner, name, tag, rec.SHA256, sha)
	}
	return nil
}

// UpdateIndex adds or updates a tag record in the index, sets active_tag, and updates HEAD.
func UpdateIndex(owner, name, tag string, sha256Hex, md5Hex string, sizeBytes int64) error {
	idx, err := LoadIndex(owner, name)
	if err != nil {
		return err
	}
	if idx == nil {
		idx = &Index{
			SchemaVersion: 2,
			Owner:         owner,
			Name:          name,
			Tags:          make(map[string]TagRecord),
		}
	}
	idx.SchemaVersion = 2
	idx.ActiveTag = tag
	idx.Tags[tag] = TagRecord{
		SHA256:    sha256Hex,
		MD5:       md5Hex,
		SizeBytes: sizeBytes,
		PulledAt:  time.Now(),
	}
	if err := SetHead(owner, name, tag); err != nil {
		return fmt.Errorf("setting HEAD: %w", err)
	}
	return SaveIndex(idx)
}

// DeleteTag removes a tag (symlink or regular file) and its index record.
// If the deleted tag was active, HEAD and ActiveTag are cleared.
// Orphaned blobs are cleaned up afterwards.
func DeleteTag(owner, name, tag string) error {
	idx, err := LoadIndex(owner, name)
	if err != nil {
		return err
	}
	if idx == nil {
		return fmt.Errorf("no index found for %s/%s", owner, name)
	}
	if _, ok := idx.Tags[tag]; !ok {
		return fmt.Errorf("tag %q not found for %s/%s", tag, owner, name)
	}

	// Remove tag file/symlink
	tagPath := TagPath(owner, name, tag)
	if err := os.Remove(tagPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing tag %q: %w", tag, err)
	}

	// Remove from index
	delete(idx.Tags, tag)

	// If this was the active tag, clear HEAD
	if idx.ActiveTag == tag {
		idx.ActiveTag = ""
		_ = os.Remove(HeadPath(owner, name))
	}

	if err := SaveIndex(idx); err != nil {
		return err
	}

	// Garbage collect orphaned blobs
	return GarbageCollectBlobs(owner, name)
}

// GarbageCollectBlobs removes blobs not referenced by any tag symlink.
func GarbageCollectBlobs(owner, name string) error {
	blobDir := BlobDir(owner, name)
	blobs, err := os.ReadDir(blobDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	// Collect referenced sha256 values from tag symlinks and HEAD
	referenced := make(map[string]bool)

	// Check HEAD — if it points to a blob, mark it as referenced
	if headRef, err := ReadHeadRef(owner, name); err == nil && headRef.Kind == HeadKindBlob {
		referenced[headRef.Value] = true
	}

	tagDir := TagDir(owner, name)
	tags, err := os.ReadDir(tagDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, entry := range tags {
		tagPath := filepath.Join(tagDir, entry.Name())
		target, err := os.Readlink(tagPath)
		if err != nil {
			// Regular file (editable tag) — read its content and hash it
			content, rerr := os.ReadFile(tagPath)
			if rerr != nil {
				continue
			}
			sha, _ := ComputeChecksums(content)
			referenced[sha] = true
			continue
		}
		// Symlink — extract the blob hash from the target path
		referenced[filepath.Base(target)] = true
	}

	// Remove unreferenced blobs
	for _, blob := range blobs {
		if blob.Name() == "" || strings.HasSuffix(blob.Name(), ".tmp") {
			continue
		}
		if !referenced[blob.Name()] {
			_ = os.Remove(filepath.Join(blobDir, blob.Name()))
		}
	}
	return nil
}

// MigrateV1ToV2 converts the flat {tag}.yaml layout to the new blob+tags+HEAD layout.
// Idempotent: if old .yaml files still exist, they get migrated; if already gone, skipped.
func MigrateV1ToV2(owner, name string) error {
	idx, err := loadIndexRaw(owner, name)
	if err != nil || idx == nil {
		return err
	}

	pipeDir := PipePath(owner, name)

	// 1. Ensure new directories exist
	if err := os.MkdirAll(BlobDir(owner, name), 0o755); err != nil {
		return fmt.Errorf("creating blob dir: %w", err)
	}
	if err := os.MkdirAll(TagDir(owner, name), 0o755); err != nil {
		return fmt.Errorf("creating tags dir: %w", err)
	}

	// 2. Migrate each tag
	for tag := range idx.Tags {
		oldPath := filepath.Join(pipeDir, tag+".yaml")
		content, err := os.ReadFile(oldPath)
		if err != nil {
			if os.IsNotExist(err) {
				// Already migrated or file missing — check if tag symlink exists
				if _, serr := os.Lstat(TagPath(owner, name, tag)); serr == nil {
					continue
				}
				continue
			}
			return fmt.Errorf("reading old tag %q: %w", tag, err)
		}

		sha, writeErr := WriteBlob(owner, name, content)
		if writeErr != nil {
			return fmt.Errorf("writing blob for tag %q: %w", tag, writeErr)
		}

		if err := CreateTagSymlink(owner, name, tag, sha); err != nil {
			return fmt.Errorf("creating symlink for tag %q: %w", tag, err)
		}

		// Update record checksum if needed
		rec := idx.Tags[tag]
		if rec.SHA256 == "" {
			recSha, recMD5 := ComputeChecksums(content)
			rec.SHA256 = recSha
			rec.MD5 = recMD5
			rec.SizeBytes = int64(len(content))
			idx.Tags[tag] = rec
		}

		// Remove old flat file
		_ = os.Remove(oldPath)
	}

	// 3. Create HEAD symlink
	activeTag := idx.ActiveTag
	if activeTag == "" && len(idx.Tags) > 0 {
		// Pick first tag alphabetically as fallback
		for t := range idx.Tags {
			if activeTag == "" || t < activeTag {
				activeTag = t
			}
		}
		idx.ActiveTag = activeTag
	}
	if activeTag != "" {
		if err := SetHead(owner, name, activeTag); err != nil {
			return fmt.Errorf("setting HEAD: %w", err)
		}
	}

	// 4. Update schema version and save
	idx.SchemaVersion = 2
	return SaveIndex(idx)
}

// HubPipeInfo holds metadata for listing hub pipes.
type HubPipeInfo struct {
	Owner     string
	Name      string
	ActiveTag string
}

// ListPipes returns all hub pipes found in ~/.pipe/hub/.
func ListPipes() ([]HubPipeInfo, error) {
	owners, err := os.ReadDir(config.HubDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var pipes []HubPipeInfo
	for _, ownerEntry := range owners {
		if !ownerEntry.IsDir() {
			continue
		}
		ownerName := ownerEntry.Name()
		names, err := os.ReadDir(filepath.Join(config.HubDir, ownerName))
		if err != nil {
			continue
		}
		for _, nameEntry := range names {
			if !nameEntry.IsDir() {
				continue
			}
			pipeName := nameEntry.Name()
			idx, err := LoadIndex(ownerName, pipeName)
			if err != nil || idx == nil {
				continue
			}
			activeTag, _ := ReadHead(ownerName, pipeName)
			if activeTag == "" {
				activeTag = idx.ActiveTag
			}
			pipes = append(pipes, HubPipeInfo{
				Owner:     ownerName,
				Name:      pipeName,
				ActiveTag: activeTag,
			})
		}
	}
	sort.Slice(pipes, func(i, j int) bool {
		ai := pipes[i].Owner + "/" + pipes[i].Name
		aj := pipes[j].Owner + "/" + pipes[j].Name
		return ai < aj
	})
	return pipes, nil
}
