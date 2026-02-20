package state

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/getpipe-dev/pipe/internal/config"
)

type RunState struct {
	RunID        string                `json:"run_id"`
	PipelineName string                `json:"pipeline_name"`
	StartedAt    time.Time             `json:"started_at"`
	FinishedAt   *time.Time            `json:"finished_at,omitempty"`
	Status       string                `json:"status"` // running|done|failed
	Steps        map[string]StepState  `json:"steps"`
}

type StepState struct {
	Status    string                `json:"status"` // pending|running|done|failed
	ExitCode  int                   `json:"exit_code"`
	Output    string                `json:"output,omitempty"`
	Sensitive bool                  `json:"sensitive"`
	At        *time.Time            `json:"at,omitempty"`
	Attempts  int                   `json:"attempts,omitempty"`
	SubSteps  map[string]StepState  `json:"sub_steps,omitempty"`
}

func NewUUID() string {
	var uuid [16]byte
	if _, err := rand.Read(uuid[:]); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	uuid[6] = (uuid[6] & 0x0f) | 0x40 // version 4
	uuid[8] = (uuid[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:16])
}

func NewRunState(pipelineName string) *RunState {
	return &RunState{
		RunID:        NewUUID(),
		PipelineName: pipelineName,
		StartedAt:    time.Now(),
		Status:       "running",
		Steps:        make(map[string]StepState),
	}
}

func statePath(pipelineName, runID string) string {
	return filepath.Join(config.StateDir, pipelineName, runID+".json")
}

func Save(rs *RunState) error {
	path := statePath(rs.PipelineName, rs.RunID)
	data, err := json.MarshalIndent(rs, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling state: %w", err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("writing state tmp: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("renaming state: %w", err)
	}
	return nil
}

func Load(pipelineName, runID string) (*RunState, error) {
	path := statePath(pipelineName, runID)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("run %q not found for pipeline %q", runID, pipelineName)
		}
		return nil, fmt.Errorf("reading state: %w", err)
	}
	var rs RunState
	if err := json.Unmarshal(data, &rs); err != nil {
		return nil, fmt.Errorf("parsing state: %w", err)
	}
	return &rs, nil
}
