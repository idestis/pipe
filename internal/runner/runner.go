package runner

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/idestis/pipe/internal/cache"
	"github.com/idestis/pipe/internal/logging"
	"github.com/idestis/pipe/internal/model"
	"github.com/idestis/pipe/internal/state"
)

type Runner struct {
	pipeline *model.Pipeline
	state    *state.RunState
	log      *logging.Logger
	envVars  map[string]string
}

func New(p *model.Pipeline, rs *state.RunState, log *logging.Logger, vars map[string]string) *Runner {
	env := make(map[string]string)
	for k, v := range vars {
		env[k] = v
	}
	return &Runner{
		pipeline: p,
		state:    rs,
		log:      log,
		envVars:  env,
	}
}

func (r *Runner) saveState() {
	if err := state.Save(r.state); err != nil {
		r.log.Log("error: failed to save state: %v", err)
	}
}

func (r *Runner) Run() error {
	for _, step := range r.pipeline.Steps {
		if err := r.runStep(step); err != nil {
			r.state.Status = "failed"
			now := time.Now()
			r.state.FinishedAt = &now
			r.saveState()
			r.log.Log("pipeline failed at step %q: %v", step.ID, err)
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
	r.log.Log("pipeline %q completed (run %s)", r.pipeline.Name, r.state.RunID)
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

// tryCache checks if a valid cache entry exists for the step.
// On hit: restores env vars, updates run state, returns true.
// On error: logs warning and returns false (graceful degradation).
func (r *Runner) tryCache(step model.Step) (bool, error) {
	if !step.Cached.Enabled {
		return false, nil
	}

	entry, err := cache.Load(step.ID)
	if err != nil {
		r.log.Log("[%s] cache warning: %v", step.ID, err)
		return false, nil
	}
	if !cache.IsValid(entry, time.Now()) {
		return false, nil
	}

	r.log.Log("[%s] cache hit", step.ID)

	// Restore env vars from cache
	if !entry.Sensitive {
		if entry.Output != "" {
			r.envVars[EnvKey(step.ID)] = strings.TrimRight(entry.Output, "\n")
		}
		for _, sub := range entry.SubOutputs {
			if !sub.Sensitive && sub.Output != "" {
				r.envVars[EnvKey(step.ID, sub.ID)] = strings.TrimRight(sub.Output, "\n")
			}
		}
	}

	// Update run state to done
	ss := r.state.Steps[step.ID]
	ss.Status = "done"
	ss.ExitCode = 0
	ss.Sensitive = step.Sensitive
	if !step.Sensitive {
		ss.Output = entry.Output
	}
	now := time.Now()
	ss.At = &now
	r.state.Steps[step.ID] = ss
	r.saveState()

	return true, nil
}

// saveCache stores a cache entry for a successfully completed step.
func (r *Runner) saveCache(step model.Step, entry *cache.Entry) {
	if !step.Cached.Enabled {
		return
	}

	now := time.Now()
	entry.CachedAt = now

	expiresAt, err := cache.ParseExpiry(step.Cached.ExpireAfter, now)
	if err != nil {
		r.log.Log("[%s] cache warning: invalid expiry %q: %v", step.ID, step.Cached.ExpireAfter, err)
		return
	}
	if !expiresAt.IsZero() {
		entry.ExpiresAt = &expiresAt
	}

	if err := cache.Save(entry); err != nil {
		r.log.Log("[%s] cache warning: %v", step.ID, err)
	}
}

func (r *Runner) runStep(step model.Step) error {
	ss := r.state.Steps[step.ID]

	// Resume logic: skip done non-sensitive steps
	if ss.Status == "done" && !step.Sensitive {
		r.log.Log("[%s] skipping (already done)", step.ID)
		return nil
	}

	// Cache check: before execution
	if hit, err := r.tryCache(step); err != nil {
		return err
	} else if hit {
		return nil
	}

	sl := r.log.Step(step.ID, step.Sensitive)
	if step.Sensitive {
		sl.Redacted()
	}

	switch {
	case step.Run.IsSingle():
		return r.runSingle(step, sl)
	case step.Run.IsStrings():
		return r.runParallelStrings(step, sl)
	case step.Run.IsSubRuns():
		return r.runParallelSubRuns(step, sl)
	default:
		return fmt.Errorf("step %q: no run command", step.ID)
	}
}

func (r *Runner) runSingle(step model.Step, sl *logging.StepLogger) error {
	ss := r.state.Steps[step.ID]
	ss.Status = "running"
	r.state.Steps[step.ID] = ss
	r.saveState()

	sl.Log("%s", step.Run.Single)

	maxAttempts := step.Retry + 1
	var output string

	attempts, err := Retry(maxAttempts, func() error {
		var execErr error
		output, execErr = r.execCapture(step.Run.Single, sl)
		return execErr
	})

	now := time.Now()
	ss.At = &now
	ss.Attempts = attempts

	if err != nil {
		code := exitCode(err)
		ss.Status = "failed"
		ss.ExitCode = code
		r.state.Steps[step.ID] = ss
		r.saveState()
		sl.Exit(code)
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
	sl.Exit(0)

	r.envVars[EnvKey(step.ID)] = strings.TrimRight(output, "\n")

	cacheOutput := output
	if step.Sensitive {
		cacheOutput = ""
	}
	r.saveCache(step, &cache.Entry{
		StepID:    step.ID,
		ExitCode:  0,
		Output:    cacheOutput,
		Sensitive: step.Sensitive,
		RunType:   "single",
	})

	return nil
}

func (r *Runner) runParallelStrings(step model.Step, sl *logging.StepLogger) error {
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
			sl.Log("parallel: %s", c)
			if err := r.execNoCapture(c, sl); err != nil {
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

	r.saveCache(step, &cache.Entry{
		StepID:  step.ID,
		RunType: "strings",
	})

	return nil
}

func (r *Runner) runParallelSubRuns(step model.Step, _ *logging.StepLogger) error {
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
			r.log.Log("[%s/%s] skipping (already done)", step.ID, sub.ID)
			continue
		}

		wg.Add(1)
		go func(sr model.SubRun) {
			defer wg.Done()
			subSl := r.log.Step(step.ID+"/"+sr.ID, sr.Sensitive)
			if sr.Sensitive {
				subSl.Redacted()
			}
			subSl.Log("%s", sr.Run)

			output, err := r.execCapture(sr.Run, subSl)

			mu.Lock()
			defer mu.Unlock()

			now := time.Now()
			subState := state.StepState{At: &now}

			if err != nil {
				code := exitCode(err)
				subState.Status = "failed"
				subState.ExitCode = code
				ss.SubSteps[sr.ID] = subState
				errs = append(errs, fmt.Sprintf("%s: %v", sr.ID, err))
				subSl.Exit(code)
			} else {
				subState.Status = "done"
				subState.ExitCode = 0
				subState.Sensitive = sr.Sensitive
				if !sr.Sensitive {
					subState.Output = output
				}
				ss.SubSteps[sr.ID] = subState
				r.envVars[EnvKey(step.ID, sr.ID)] = strings.TrimRight(output, "\n")
				subSl.Exit(0)
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

	// Build sub-outputs for cache
	var subOutputs []cache.SubEntry
	for _, sr := range step.Run.SubRuns {
		sub := ss.SubSteps[sr.ID]
		subOutputs = append(subOutputs, cache.SubEntry{
			ID:        sr.ID,
			Output:    sub.Output,
			Sensitive: sub.Sensitive,
			ExitCode:  sub.ExitCode,
		})
	}
	r.saveCache(step, &cache.Entry{
		StepID:     step.ID,
		Sensitive:  step.Sensitive,
		RunType:    "subruns",
		SubOutputs: subOutputs,
	})

	return nil
}

func (r *Runner) execCapture(cmdStr string, sl *logging.StepLogger) (string, error) {
	cmd := exec.Command("sh", "-c", cmdStr)
	cmd.Env = BuildEnv(r.envVars)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = sl.Writer()
	err := cmd.Run()
	return stdout.String(), err
}

func (r *Runner) execNoCapture(cmdStr string, sl *logging.StepLogger) error {
	cmd := exec.Command("sh", "-c", cmdStr)
	cmd.Env = BuildEnv(r.envVars)
	cmd.Stdout = sl.Writer()
	cmd.Stderr = sl.Writer()
	return cmd.Run()
}

func exitCode(err error) int {
	if ee, ok := err.(*exec.ExitError); ok {
		return ee.ExitCode()
	}
	return 1
}
