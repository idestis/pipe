package ui

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/idestis/pipe/internal/model"
)

// Status represents the current state of a pipeline step.
type Status int

const (
	Waiting Status = iota // ○
	Running               // ●
	Done                  // ✓
	Failed                // ✗
)

// ANSI color helpers
const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorDim    = "\033[2m"
)

var icons = [...]string{
	Waiting: colorDim + "○" + colorReset,
	Running: colorYellow + "●" + colorReset,
	Done:    colorGreen + "✓" + colorReset,
	Failed:  colorRed + "✗" + colorReset,
}

type row struct {
	id        string
	status    Status
	startedAt time.Time
	duration  time.Duration
}

// StatusUI renders a compact, live-updating status display to a terminal.
type StatusUI struct {
	mu       sync.Mutex
	w        io.Writer
	rows     []row
	index    map[string]int // id → rows index
	lines    int            // lines rendered last frame (for cursor-up)
	maxWidth int            // longest id (for column alignment)
}

// NewStatusUI creates a StatusUI from the pipeline steps.
// Parallel sub-items are expanded eagerly.
func NewStatusUI(w io.Writer, steps []model.Step) *StatusUI {
	s := &StatusUI{
		w:     w,
		index: make(map[string]int),
	}

	for _, step := range steps {
		switch {
		case step.Run.IsStrings():
			for i := range step.Run.Strings {
				id := fmt.Sprintf("%s/run_%d", step.ID, i)
				s.addRow(id)
			}
		case step.Run.IsSubRuns():
			for _, sub := range step.Run.SubRuns {
				id := fmt.Sprintf("%s/%s", step.ID, sub.ID)
				s.addRow(id)
			}
		default:
			s.addRow(step.ID)
		}
	}

	return s
}

func (s *StatusUI) addRow(id string) {
	s.index[id] = len(s.rows)
	s.rows = append(s.rows, row{id: id, status: Waiting})
	if len(id) > s.maxWidth {
		s.maxWidth = len(id)
	}
}

// SetStatus updates the status of a step and re-renders.
func (s *StatusUI) SetStatus(id string, st Status) {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx, ok := s.index[id]
	if !ok {
		return
	}
	r := &s.rows[idx]
	r.status = st

	switch st {
	case Running:
		r.startedAt = time.Now()
	case Done, Failed:
		if !r.startedAt.IsZero() {
			r.duration = time.Since(r.startedAt)
		}
	}

	s.render()
}

// Finish performs a final render. No subsequent redraws occur.
func (s *StatusUI) Finish() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.render()
}

// render draws all rows, overwriting the previous frame.
// Must be called with s.mu held.
func (s *StatusUI) render() {
	// Move cursor up to overwrite previous frame
	if s.lines > 0 {
		_, _ = fmt.Fprintf(s.w, "\033[%dA", s.lines)
	}

	for _, r := range s.rows {
		icon := icons[r.status]
		suffix := statusSuffix(r)
		// \033[2K clears the entire line
		_, _ = fmt.Fprintf(s.w, "\033[2K%s %-*s  %s\n", icon, s.maxWidth, r.id, suffix)
	}

	s.lines = len(s.rows)
}

func statusSuffix(r row) string {
	switch r.status {
	case Waiting:
		return colorDim + "waiting" + colorReset
	case Running:
		return colorYellow + "running..." + colorReset
	case Done:
		return colorDim + formatDuration(r.duration) + colorReset
	case Failed:
		return colorRed + formatDuration(r.duration) + colorReset
	default:
		return ""
	}
}

// formatDuration returns a human-friendly duration string.
func formatDuration(d time.Duration) string {
	secs := d.Seconds()
	if secs < 60 {
		return fmt.Sprintf("(%.1fs)", secs)
	}
	m := int(secs) / 60
	s := int(secs) % 60
	return fmt.Sprintf("(%dm %ds)", m, s)
}

// IsTTY reports whether f is connected to a terminal.
func IsTTY(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
