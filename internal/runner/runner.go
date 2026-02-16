package runner

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/destis/pipe/internal/logging"
	"github.com/destis/pipe/internal/model"
	"github.com/destis/pipe/internal/state"
)

type Runner struct {
	pipeline *model.Pipeline
	state    *state.RunState
	log      *logging.Logger
	envVars  map[string]string
}

func New(p *model.Pipeline, rs *state.RunState, log *logging.Logger) *Runner {
	return &Runner{
		pipeline: p,
		state:    rs,
		log:      log,
		envVars:  make(map[string]string),
	}
}

func (r *Runner) saveState() {
	if err := state.Save(r.state); err != nil {
		r.log.Error("failed to save state: %v", err)
	}
}

func (r *Runner) Run() error {
	for _, step := range r.pipeline.Steps {
		if err := r.runStep(step); err != nil {
			r.state.Status = "failed"
			now := time.Now()
			r.state.FinishedAt = &now
			r.saveState()
			r.log.Error("pipeline failed at step %q: %v", step.ID, err)
			fmt.Fprintf(os.Stderr,
				"\nPipeline failed. Resume with:\n  pipe %s --resume %s\n\n",
				r.pipeline.Name, r.state.RunID,
			)
			return err
		}
	}

	r.state.Status = "done"
	now := time.Now()
	r.state.FinishedAt = &now
	r.saveState()
	r.log.Info("pipeline %q completed (run %s)", r.pipeline.Name, r.state.RunID)
	return nil
}

// RestoreEnvFromState rebuilds the env map from a previous run's completed steps.
func (r *Runner) RestoreEnvFromState() {
	for _, step := range r.pipeline.Steps {
		ss, ok := r.state.Steps[step.ID]
		if !ok {
			continue
		}
		if ss.Status == "done" && !ss.Sensitive {
			if ss.Output != "" {
				r.envVars[EnvKey(step.ID)] = strings.TrimRight(ss.Output, "\n")
			}
			for subID, sub := range ss.SubSteps {
				if sub.Status == "done" && !sub.Sensitive && sub.Output != "" {
					r.envVars[EnvKey(step.ID, subID)] = strings.TrimRight(sub.Output, "\n")
				}
			}
		}
	}
}

func (r *Runner) runStep(step model.Step) error {
	ss := r.state.Steps[step.ID]

	// Resume logic: skip done non-sensitive steps
	if ss.Status == "done" && !step.Sensitive {
		r.log.Info("[%s] skipping (already done)", step.ID)
		return nil
	}

	r.log.Info("[%s] running", step.ID)

	switch {
	case step.Run.IsSingle():
		return r.runSingle(step)
	case step.Run.IsStrings():
		return r.runParallelStrings(step)
	case step.Run.IsSubRuns():
		return r.runParallelSubRuns(step)
	default:
		return fmt.Errorf("step %q: no run command", step.ID)
	}
}

func (r *Runner) runSingle(step model.Step) error {
	ss := r.state.Steps[step.ID]
	ss.Status = "running"
	r.state.Steps[step.ID] = ss
	r.saveState()

	maxAttempts := step.Retry + 1
	var output string

	attempts, err := Retry(maxAttempts, func() error {
		var execErr error
		output, execErr = r.execCapture(step.Run.Single)
		return execErr
	})

	now := time.Now()
	ss.At = &now
	ss.Attempts = attempts

	if err != nil {
		ss.Status = "failed"
		ss.ExitCode = exitCode(err)
		r.state.Steps[step.ID] = ss
		r.saveState()
		return fmt.Errorf("step %q failed: %w", step.ID, err)
	}

	ss.Status = "done"
	ss.ExitCode = 0
	ss.Sensitive = step.Sensitive
	if !step.Sensitive {
		ss.Output = output
	}
	r.state.Steps[step.ID] = ss
	r.saveState()

	r.envVars[EnvKey(step.ID)] = strings.TrimRight(output, "\n")
	return nil
}

func (r *Runner) runParallelStrings(step model.Step) error {
	ss := r.state.Steps[step.ID]
	ss.Status = "running"
	r.state.Steps[step.ID] = ss
	r.saveState()

	var (
		mu   sync.Mutex
		errs []string
		wg   sync.WaitGroup
	)

	for _, cmd := range step.Run.Strings {
		wg.Add(1)
		go func(c string) {
			defer wg.Done()
			r.log.Info("[%s] parallel: %s", step.ID, c)
			if err := r.execNoCapture(c); err != nil {
				mu.Lock()
				errs = append(errs, fmt.Sprintf("%s: %v", c, err))
				mu.Unlock()
			}
		}(cmd)
	}
	wg.Wait()

	now := time.Now()
	ss.At = &now

	if len(errs) > 0 {
		ss.Status = "failed"
		r.state.Steps[step.ID] = ss
		r.saveState()
		return fmt.Errorf("step %q parallel failures: %s", step.ID, strings.Join(errs, "; "))
	}

	ss.Status = "done"
	ss.ExitCode = 0
	r.state.Steps[step.ID] = ss
	r.saveState()
	return nil
}

func (r *Runner) runParallelSubRuns(step model.Step) error {
	ss := r.state.Steps[step.ID]
	ss.Status = "running"
	if ss.SubSteps == nil {
		ss.SubSteps = make(map[string]state.StepState)
	}
	r.state.Steps[step.ID] = ss
	r.saveState()

	var (
		mu   sync.Mutex
		errs []string
		wg   sync.WaitGroup
	)

	for _, sub := range step.Run.SubRuns {
		existing := ss.SubSteps[sub.ID]
		// Resume: skip done non-sensitive sub-runs
		if existing.Status == "done" && !sub.Sensitive {
			r.log.Info("[%s/%s] skipping (already done)", step.ID, sub.ID)
			continue
		}

		wg.Add(1)
		go func(sr model.SubRun) {
			defer wg.Done()
			r.log.Info("[%s/%s] running", step.ID, sr.ID)

			output, err := r.execCapture(sr.Run)

			mu.Lock()
			defer mu.Unlock()

			now := time.Now()
			subState := state.StepState{At: &now}

			if err != nil {
				subState.Status = "failed"
				subState.ExitCode = exitCode(err)
				ss.SubSteps[sr.ID] = subState
				errs = append(errs, fmt.Sprintf("%s: %v", sr.ID, err))
			} else {
				subState.Status = "done"
				subState.ExitCode = 0
				subState.Sensitive = sr.Sensitive
				if !sr.Sensitive {
					subState.Output = output
				}
				ss.SubSteps[sr.ID] = subState
				r.envVars[EnvKey(step.ID, sr.ID)] = strings.TrimRight(output, "\n")
			}
		}(sub)
	}
	wg.Wait()

	now := time.Now()
	ss.At = &now

	if len(errs) > 0 {
		ss.Status = "failed"
		r.state.Steps[step.ID] = ss
		r.saveState()
		return fmt.Errorf("step %q sub-run failures: %s", step.ID, strings.Join(errs, "; "))
	}

	ss.Status = "done"
	ss.ExitCode = 0
	r.state.Steps[step.ID] = ss
	r.saveState()
	return nil
}

func (r *Runner) execCapture(cmdStr string) (string, error) {
	cmd := exec.Command("sh", "-c", cmdStr)
	cmd.Env = BuildEnv(r.envVars)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = newLogWriter(r.log)
	err := cmd.Run()
	return stdout.String(), err
}

func (r *Runner) execNoCapture(cmdStr string) error {
	cmd := exec.Command("sh", "-c", cmdStr)
	cmd.Env = BuildEnv(r.envVars)
	cmd.Stdout = newLogWriter(r.log)
	cmd.Stderr = newLogWriter(r.log)
	return cmd.Run()
}

func exitCode(err error) int {
	if ee, ok := err.(*exec.ExitError); ok {
		return ee.ExitCode()
	}
	return 1
}

// logWriter sends command output lines to the logger.
type logWriter struct {
	log *logging.Logger
}

func newLogWriter(l *logging.Logger) *logWriter {
	return &logWriter{log: l}
}

func (w *logWriter) Write(p []byte) (int, error) {
	s := strings.TrimRight(string(p), "\n")
	if s != "" {
		w.log.Info("  | %s", s)
	}
	return len(p), nil
}

