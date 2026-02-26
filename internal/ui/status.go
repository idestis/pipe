package ui

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/getpipe-dev/pipe/internal/model"
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
	output    []string // collected output, shown only after step finishes
	flushed   bool     // true after output has been flushed to history
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
		if step.Interactive {
			continue
		}
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
// When transitioning to Done/Failed, any collected output is flushed
// above the status block with a colored pipe prefix.
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
		// Flush collected output above the status block
		if len(r.output) > 0 {
			s.flushOutput(r)
		}
	}

	s.render()
}

// flushOutput prints collected output above the status block, then clears it.
// Before flushing the target row, any preceding completed rows that haven't
// been flushed yet are also flushed so terminal scrollback preserves the
// original pipeline order.
// Must be called with s.mu held.
func (s *StatusUI) flushOutput(r *row) {
	if s.lines > 0 {
		_, _ = fmt.Fprintf(s.w, "\033[%dA", s.lines)
	}

	// Flush preceding completed rows to preserve pipeline order.
	targetIdx := s.index[r.id]
	for i := 0; i < targetIdx; i++ {
		prev := &s.rows[i]
		if prev.flushed || (prev.status != Done && prev.status != Failed) {
			continue
		}
		s.flushRow(prev)
	}

	s.flushRow(r)
	s.lines = 0
}

// flushRow writes a single row's status line and any collected output, then
// marks it as flushed. Must be called with s.mu held.
func (s *StatusUI) flushRow(r *row) {
	icon := icons[r.status]
	suffix := statusSuffix(*r)
	_, _ = fmt.Fprintf(s.w, "\033[2K%s %-*s  %s\n", icon, s.maxWidth, r.id, suffix)

	if len(r.output) > 0 {
		pipe := outputPipe(r.status)
		for _, line := range r.output {
			_, _ = fmt.Fprintf(s.w, "\033[2K%s %s\n", pipe, line)
		}
	}

	r.output = nil
	r.flushed = true
}

// PrintAbove prints output line(s) above the status block, then re-renders
// status rows below. The output scrolls into terminal history while the
// status block stays pinned at the bottom.
func (s *StatusUI) PrintAbove(msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Move cursor up to the top of the status block
	if s.lines > 0 {
		_, _ = fmt.Fprintf(s.w, "\033[%dA", s.lines)
	}

	// Print each line of the message, clearing the line first
	for _, line := range strings.Split(msg, "\n") {
		_, _ = fmt.Fprintf(s.w, "\033[2K%s\n", line)
	}

	// Reset lines and re-render the status block below
	s.lines = 0
	s.render()
}

// AddOutput appends a line of output to the given step row.
// Output is collected but only rendered after the step finishes (Done/Failed).
func (s *StatusUI) AddOutput(id string, line string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx, ok := s.index[id]
	if !ok {
		return
	}
	s.rows[idx].output = append(s.rows[idx].output, line)
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

	n := 0
	for _, r := range s.rows {
		if r.flushed {
			continue
		}
		icon := icons[r.status]
		suffix := statusSuffix(r)
		// \033[2K clears the entire line
		_, _ = fmt.Fprintf(s.w, "\033[2K%s %-*s  %s\n", icon, s.maxWidth, r.id, suffix)
		n++
	}

	s.lines = n
}

func outputPipe(s Status) string {
	switch s {
	case Done:
		return colorGreen + "│" + colorReset
	case Failed:
		return colorRed + "│" + colorReset
	default:
		return colorDim + "│" + colorReset
	}
}

func statusSuffix(r row) string {
	switch r.status {
	case Waiting:
		return colorDim + "waiting" + colorReset
	case Running:
		return colorYellow + "running..." + colorReset
	case Done:
		return colorDim + FormatDuration(r.duration) + colorReset
	case Failed:
		return colorRed + FormatDuration(r.duration) + colorReset
	default:
		return ""
	}
}

// FormatDuration returns a human-friendly duration string.
func FormatDuration(d time.Duration) string {
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
