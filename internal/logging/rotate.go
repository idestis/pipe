package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"github.com/charmbracelet/log"
	"github.com/getpipe-dev/pipe/internal/config"
)

// RotateLogs removes old log files for the given pipeline, keeping the newest
// N files (default 10, controlled by PIPE_LOG_ROTATE). Setting the env var to
// 0 disables rotation.
func RotateLogs(pipelineName string) error {
	limit := config.ParseRotateEnv("PIPE_LOG_ROTATE", 10)
	if limit == 0 {
		return nil
	}

	// Compute log directory and base name the same way logging.New does.
	// For hub pipes like "owner/name", logs live in ~/.pipe/logs/owner/
	// with filenames like "name-{rid}-{ts}.log".
	logDir := filepath.Join(config.LogDir, filepath.Dir(pipelineName))
	base := filepath.Base(pipelineName)

	entries, err := os.ReadDir(logDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("reading log directory: %w", err)
	}

	// Match only log files for this exact pipeline base name.
	// Pattern: {base}-{8hex}-{YYYYMMDD}-{HHMMSS}.log
	pattern := regexp.MustCompile(`^` + regexp.QuoteMeta(base) + `-[a-f0-9]{8}-\d{8}-\d{6}\.log$`)

	type logEntry struct {
		name    string
		modTime int64
	}
	var matched []logEntry
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !pattern.MatchString(e.Name()) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		matched = append(matched, logEntry{name: e.Name(), modTime: info.ModTime().UnixNano()})
	}

	if len(matched) <= limit {
		return nil
	}

	// Sort newest-first by modification time.
	sort.Slice(matched, func(i, j int) bool {
		return matched[i].modTime > matched[j].modTime
	})

	// Delete everything beyond the keep limit.
	for _, entry := range matched[limit:] {
		path := filepath.Join(logDir, entry.name)
		if err := os.Remove(path); err != nil {
			log.Warn("failed to remove old log file", "path", path, "err", err)
		} else {
			log.Debug("rotated old log file", "path", path)
		}
	}

	return nil
}
