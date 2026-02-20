package state

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/getpipe-dev/pipe/internal/config"
)

// RotateStates removes old state files for the given pipeline, keeping the
// newest N files (default 10, controlled by PIPE_STATE_ROTATE). The current
// run's state file is never deleted. Setting the env var to 0 disables
// rotation.
func RotateStates(pipelineName, currentRunID string) error {
	limit := config.ParseRotateEnv("PIPE_STATE_ROTATE", 10)
	if limit == 0 {
		return nil
	}

	stateDir := filepath.Join(config.StateDir, pipelineName)
	entries, err := os.ReadDir(stateDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("reading state directory: %w", err)
	}

	currentFile := currentRunID + ".json"

	type stateEntry struct {
		name    string
		modTime int64
	}
	var candidates []stateEntry
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		// Skip non-JSON files and tmp files
		if !strings.HasSuffix(name, ".json") || strings.HasSuffix(name, ".tmp") {
			continue
		}
		// Never consider the current run for deletion
		if name == currentFile {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		candidates = append(candidates, stateEntry{name: name, modTime: info.ModTime().UnixNano()})
	}

	// Current run occupies one slot in the limit.
	keepOthers := max(limit-1, 0)

	if len(candidates) <= keepOthers {
		return nil
	}

	// Sort newest-first by modification time.
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].modTime > candidates[j].modTime
	})

	// Delete everything beyond the keep limit.
	for _, entry := range candidates[keepOthers:] {
		path := filepath.Join(stateDir, entry.name)
		if err := os.Remove(path); err != nil {
			log.Warn("failed to remove old state file", "path", path, "err", err)
		} else {
			log.Debug("rotated old state file", "path", path)
		}
	}

	return nil
}
