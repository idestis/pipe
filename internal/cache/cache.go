package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/getpipe-dev/pipe/internal/config"
)

// Entry represents a cached step result.
type Entry struct {
	StepID     string      `json:"step_id"`
	CachedAt   time.Time   `json:"cached_at"`
	ExpiresAt  *time.Time  `json:"expires_at,omitempty"`
	ExitCode   int         `json:"exit_code"`
	Output     string      `json:"output,omitempty"`
	Sensitive  bool        `json:"sensitive"`
	SubOutputs []SubEntry  `json:"sub_outputs,omitempty"`
	RunType    string      `json:"run_type"` // single, strings, subruns
}

// SubEntry stores per-sub-run cached output.
type SubEntry struct {
	ID        string `json:"id"`
	Output    string `json:"output,omitempty"`
	Sensitive bool   `json:"sensitive"`
	ExitCode  int    `json:"exit_code"`
}

func cachePath(stepID string) string {
	return filepath.Join(config.CacheDir, stepID+".json")
}

// Save writes a cache entry atomically (tmp + rename).
func Save(entry *Entry) error {
	path := cachePath(entry.StepID)
	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling cache: %w", err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("writing cache tmp: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("renaming cache: %w", err)
	}
	return nil
}

// Load reads a cache entry by step ID.
// Returns nil, nil if the file does not exist (not an error).
func Load(stepID string) (*Entry, error) {
	path := cachePath(stepID)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading cache: %w", err)
	}
	var entry Entry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, fmt.Errorf("parsing cache for %q: %w", stepID, err)
	}
	return &entry, nil
}

// IsValid checks whether a cache entry is still valid at the given time.
func IsValid(entry *Entry, now time.Time) bool {
	if entry == nil {
		return false
	}
	// No expiry means cache forever
	if entry.ExpiresAt == nil {
		return true
	}
	return now.Before(*entry.ExpiresAt)
}

// Clear removes the cache entry for a specific step.
func Clear(stepID string) error {
	path := cachePath(stepID)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("clearing cache for %q: %w", stepID, err)
	}
	return nil
}

// ClearAll removes all cache entries.
func ClearAll() error {
	entries, err := os.ReadDir(config.CacheDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("reading cache dir: %w", err)
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		path := filepath.Join(config.CacheDir, e.Name())
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("removing %s: %w", e.Name(), err)
		}
	}
	return nil
}

// List returns all cache entries.
func List() ([]*Entry, error) {
	entries, err := os.ReadDir(config.CacheDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading cache dir: %w", err)
	}

	var result []*Entry
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		stepID := strings.TrimSuffix(e.Name(), ".json")
		entry, err := Load(stepID)
		if err != nil {
			continue // skip corrupt entries
		}
		if entry != nil {
			result = append(result, entry)
		}
	}
	return result, nil
}
