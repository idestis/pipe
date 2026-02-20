package runner

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/getpipe-dev/pipe/internal/cache"
	"github.com/getpipe-dev/pipe/internal/graph"
	"github.com/getpipe-dev/pipe/internal/logging"
	"github.com/getpipe-dev/pipe/internal/model"
	"github.com/getpipe-dev/pipe/internal/state"
	"github.com/getpipe-dev/pipe/internal/ui"
)

type Runner struct {
	pipeline  *model.Pipeline
	state     *state.RunState
	log       *logging.Logger
	envVars   map[string]string
	ui        *ui.StatusUI // nil in verbose mode
	verbosity int
	envMu     sync.Mutex // protects envVars
	stateMu   sync.Mutex // protects state.Steps and saveState
	emitMu    sync.Mutex // protects verbose-mode stderr output
}

func New(p *model.Pipeline, rs *state.RunState, log *logging.Logger, vars map[string]string, statusUI *ui.StatusUI, verbosity int) *Runner {
	env := make(map[string]string)
	for k, v := range vars {
		env[k] = v
	}
	return &Runner{
		pipeline:  p,
		state:     rs,
		log:       log,
		envVars:   env,
		ui:        statusUI,
		verbosity: verbosity,
	}
}

// shouldShowOutput determines if a step's stdout should be shown in real-time.
//
//	| Mode              | output: true | output: false (default) |
//	|-------------------|--------------|-------------------------|
//	| v=0 (compact+TTY) | show         | no                      |
//	| v=1 (-v)          | show         | no                      |
//	| v=2 (-vv)         | show         | override: show anyway   |
//
// sensitive: true always wins — never show output.
func shouldShowOutput(step model.Step, sensitive bool, verbosity int) bool {
	if sensitive {
		return false
	}
	if verbosity >= 2 {
		return true
	}
	return step.Output
}

// outputEmitter returns an emit function and a flush function for step output.
// In compact mode (StatusUI present), emit collects output for display after
// the step finishes; flush is a no-op.
// In verbose mode, emit buffers lines and flush writes them as a grouped block
// to stderr with [stepID] prefix — preventing interleaved output from parallel steps.
func (r *Runner) outputEmitter(stepID string) (emit func(string), flush func()) {
	if r.ui != nil {
		return func(line string) {
			r.ui.AddOutput(stepID, line)
		}, func() {}
	}
	var mu sync.Mutex
	var lines []string
	return func(line string) {
		mu.Lock()
		lines = append(lines, line)
		mu.Unlock()
	}, func() {
		mu.Lock()
		defer mu.Unlock()
		if len(lines) == 0 {
			return
		}
		r.emitMu.Lock()
		for _, line := range lines {
			fmt.Fprintf(os.Stderr, "[%s] %s\n", stepID, line)
		}
		r.emitMu.Unlock()
	}
}

func (r *Runner) uiStatus(id string, s ui.Status) {
	if r.ui != nil {
		r.ui.SetStatus(id, s)
	}
}

// uiStatusStep sets the UI status for all rows belonging to a step.
func (r *Runner) uiStatusStep(step model.Step, s ui.Status) {
	if r.ui == nil {
		return
	}
	switch {
	case step.Run.IsStrings():
		for i := range step.Run.Strings {
			r.ui.SetStatus(fmt.Sprintf("%s/run_%d", step.ID, i), s)
		}
	case step.Run.IsSubRuns():
		for _, sub := range step.Run.SubRuns {
			r.ui.SetStatus(fmt.Sprintf("%s/%s", step.ID, sub.ID), s)
		}
	default:
		r.ui.SetStatus(step.ID, s)
	}
}

func (r *Runner) saveState() {
	if err := state.Save(r.state); err != nil {
		r.log.Log("error: failed to save state: %v", err)
	}
}

func (r *Runner) setStepState(id string, ss state.StepState) {
	r.stateMu.Lock()
	defer r.stateMu.Unlock()
	r.state.Steps[id] = ss
	r.saveState()
}

func (r *Runner) getStepState(id string) state.StepState {
	r.stateMu.Lock()
	defer r.stateMu.Unlock()
	return r.state.Steps[id]
}

func (r *Runner) setEnv(key, value string) {
	r.envMu.Lock()
	defer r.envMu.Unlock()
	r.envVars[key] = value
}

func (r *Runner) buildEnv() []string {
	r.envMu.Lock()
	defer r.envMu.Unlock()
	return BuildEnv(r.envVars)
}

// stepProcessCount returns the number of concurrent processes a step will spawn.
func stepProcessCount(step model.Step) int {
	switch {
	case step.Run.IsStrings():
		return len(step.Run.Strings)
	case step.Run.IsSubRuns():
		return len(step.Run.SubRuns)
	default:
		return 1
	}
}

type stepResult struct {
	ID  string
	Err error
}

func (r *Runner) Run() error {
	g, err := graph.Build(r.pipeline.Steps)
	if err != nil {
		return fmt.Errorf("building dependency graph: %w", err)
	}

	maxParallel := runtime.NumCPU()
	if v := os.Getenv("PIPE_MAX_PARALLEL"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxParallel = n
		}
	}

	// Build step lookup
	stepByID := make(map[string]model.Step)
	for _, s := range r.pipeline.Steps {
		stepByID[s.ID] = s
	}

	// Working copy of in-degree
	inDeg := make(map[string]int)
	for id, d := range g.InDegree {
		inDeg[id] = d
	}

	total := len(g.Order)
	results := make(chan stepResult, total)
	sem := make(chan struct{}, maxParallel)
	completed := 0
	failed := make(map[string]bool)
	var firstErr error

	// Seed ready steps (in-degree == 0)
	for _, id := range g.Order {
		if inDeg[id] == 0 {
			step := stepByID[id]
			go r.workerRun(step, sem, results)
		}
	}

	// Dispatch loop
	for completed < total {
		res := <-results
		completed++

		if res.Err != nil {
			failed[res.ID] = true
			if firstErr == nil {
				firstErr = res.Err
			}
			// Cascade-fail all transitive dependents
			r.cascadeFail(res.ID, g, failed, &completed)
		} else {
			// Decrement in-degree of dependents, enqueue newly-ready
			for _, dep := range g.Dependents[res.ID] {
				if failed[dep] {
					continue
				}
				inDeg[dep]--
				if inDeg[dep] == 0 {
					step := stepByID[dep]
					go r.workerRun(step, sem, results)
				}
			}
		}
	}

	if firstErr != nil {
		r.stateMu.Lock()
		r.state.Status = "failed"
		now := time.Now()
		r.state.FinishedAt = &now
		r.saveState()
		r.stateMu.Unlock()

		r.log.Log("pipeline failed: %v", firstErr)
		if r.ui != nil {
			r.ui.Finish()
		}
		fmt.Fprintf(os.Stderr,
			"\n\033[2mPipeline failed. Resume with:\n  pipe %s --resume %s\033[0m\n\n",
			r.pipeline.Name, r.state.RunID,
		)
		return firstErr
	}

	r.stateMu.Lock()
	r.state.Status = "done"
	now := time.Now()
	r.state.FinishedAt = &now
	r.saveState()
	r.stateMu.Unlock()

	r.log.Log("pipeline %q completed (run %s)", r.pipeline.Name, r.state.RunID)
	if r.ui != nil {
		r.ui.Finish()
	}
	return nil
}

// cascadeFail marks all transitive dependents of a failed step as failed.
func (r *Runner) cascadeFail(failedID string, g *graph.Graph, failedSet map[string]bool, completed *int) {
	// BFS through dependents
	queue := []string{failedID}
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		for _, dep := range g.Dependents[curr] {
			if failedSet[dep] {
				continue
			}
			failedSet[dep] = true
			r.log.Log("[%s] skipped (dependency %q failed)", dep, failedID)
			r.uiStatusStep(findStep(r.pipeline.Steps, dep), ui.Failed)

			// Mark in state
			r.stateMu.Lock()
			ss := r.state.Steps[dep]
			ss.Status = "failed"
			now := time.Now()
			ss.At = &now
			r.state.Steps[dep] = ss
			r.saveState()
			r.stateMu.Unlock()

			*completed++
			queue = append(queue, dep)
		}
	}
}

func findStep(steps []model.Step, id string) model.Step {
	for _, s := range steps {
		if s.ID == id {
			return s
		}
	}
	return model.Step{ID: id}
}

// workerRun acquires semaphore slots, runs the step, and sends the result.
func (r *Runner) workerRun(step model.Step, sem chan struct{}, results chan<- stepResult) {
	slots := stepProcessCount(step)
	for i := 0; i < slots; i++ {
		sem <- struct{}{}
	}
	err := r.runStep(step)
	for i := 0; i < slots; i++ {
		<-sem
	}
	results <- stepResult{ID: step.ID, Err: err}
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
			r.setEnv(EnvKey(step.ID), strings.TrimRight(entry.Output, "\n"))
		}
		for _, sub := range entry.SubOutputs {
			if !sub.Sensitive && sub.Output != "" {
				r.setEnv(EnvKey(step.ID, sub.ID), strings.TrimRight(sub.Output, "\n"))
			}
		}
	}

	// Update run state to done
	r.stateMu.Lock()
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
	r.stateMu.Unlock()

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
	ss := r.getStepState(step.ID)

	// Resume logic: skip done non-sensitive steps
	if ss.Status == "done" && !step.Sensitive {
		r.log.Log("[%s] skipping (already done)", step.ID)
		r.uiStatusStep(step, ui.Done)
		return nil
	}

	// Cache check: before execution
	if hit, err := r.tryCache(step); err != nil {
		return err
	} else if hit {
		r.uiStatusStep(step, ui.Done)
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
	ss := r.getStepState(step.ID)
	ss.Status = "running"
	r.setStepState(step.ID, ss)
	r.uiStatus(step.ID, ui.Running)

	sl.Log("%s", step.Run.Single)

	show := shouldShowOutput(step, step.Sensitive, r.verbosity)
	maxAttempts := step.Retry + 1
	var output string

	attempts, err := Retry(maxAttempts, func() error {
		var execErr error
		output, execErr = r.execCapture(step.Run.Single, sl, show, step.ID)
		return execErr
	})

	now := time.Now()
	ss.At = &now
	ss.Attempts = attempts

	if err != nil {
		code := exitCode(err)
		ss.Status = "failed"
		ss.ExitCode = code
		r.setStepState(step.ID, ss)
		sl.Exit(code)
		r.uiStatus(step.ID, ui.Failed)
		return fmt.Errorf("step %q failed: %w", step.ID, err)
	}

	ss.Status = "done"
	ss.ExitCode = 0
	ss.Sensitive = step.Sensitive
	if !step.Sensitive {
		ss.Output = output
	}
	r.setStepState(step.ID, ss)
	sl.Exit(0)
	r.uiStatus(step.ID, ui.Done)

	r.setEnv(EnvKey(step.ID), strings.TrimRight(output, "\n"))

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
	ss := r.getStepState(step.ID)
	ss.Status = "running"
	r.setStepState(step.ID, ss)

	var (
		mu   sync.Mutex
		errs []string
		wg   sync.WaitGroup
	)

	show := shouldShowOutput(step, step.Sensitive, r.verbosity)

	for i, cmd := range step.Run.Strings {
		wg.Add(1)
		go func(idx int, c string) {
			defer wg.Done()
			rowID := fmt.Sprintf("%s/run_%d", step.ID, idx)
			r.uiStatus(rowID, ui.Running)
			sl.Log("parallel: %s", c)
			if err := r.execNoCapture(c, sl, show, rowID); err != nil {
				mu.Lock()
				errs = append(errs, fmt.Sprintf("%s: %v", c, err))
				mu.Unlock()
				r.uiStatus(rowID, ui.Failed)
			} else {
				r.uiStatus(rowID, ui.Done)
			}
		}(i, cmd)
	}
	wg.Wait()

	now := time.Now()
	ss.At = &now

	if len(errs) > 0 {
		ss.Status = "failed"
		r.setStepState(step.ID, ss)
		return fmt.Errorf("step %q parallel failures: %s", step.ID, strings.Join(errs, "; "))
	}

	ss.Status = "done"
	ss.ExitCode = 0
	r.setStepState(step.ID, ss)

	r.saveCache(step, &cache.Entry{
		StepID:  step.ID,
		RunType: "strings",
	})

	return nil
}

func (r *Runner) runParallelSubRuns(step model.Step, _ *logging.StepLogger) error {
	r.stateMu.Lock()
	ss := r.state.Steps[step.ID]
	ss.Status = "running"
	if ss.SubSteps == nil {
		ss.SubSteps = make(map[string]state.StepState)
	}
	r.state.Steps[step.ID] = ss
	r.saveState()
	r.stateMu.Unlock()

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
			r.uiStatus(step.ID+"/"+sub.ID, ui.Done)
			continue
		}

		wg.Add(1)
		go func(sr model.SubRun) {
			defer wg.Done()
			rowID := step.ID + "/" + sr.ID
			r.uiStatus(rowID, ui.Running)
			subSl := r.log.Step(rowID, sr.Sensitive)
			if sr.Sensitive {
				subSl.Redacted()
			}
			subSl.Log("%s", sr.Run)

			show := shouldShowOutput(step, sr.Sensitive, r.verbosity)
			output, err := r.execCapture(sr.Run, subSl, show, rowID)

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
				r.uiStatus(rowID, ui.Failed)
			} else {
				subState.Status = "done"
				subState.ExitCode = 0
				subState.Sensitive = sr.Sensitive
				if !sr.Sensitive {
					subState.Output = output
				}
				ss.SubSteps[sr.ID] = subState
				r.setEnv(EnvKey(step.ID, sr.ID), strings.TrimRight(output, "\n"))
				subSl.Exit(0)
				r.uiStatus(rowID, ui.Done)
			}
		}(sub)
	}
	wg.Wait()

	now := time.Now()
	ss.At = &now

	if len(errs) > 0 {
		ss.Status = "failed"
		r.setStepState(step.ID, ss)
		return fmt.Errorf("step %q sub-run failures: %s", step.ID, strings.Join(errs, "; "))
	}

	ss.Status = "done"
	ss.ExitCode = 0
	r.setStepState(step.ID, ss)

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

func (r *Runner) execCapture(cmdStr string, sl *logging.StepLogger, showOutput bool, stepID string) (string, error) {
	cmd := exec.Command("sh", "-c", cmdStr)
	cmd.Env = r.buildEnv()
	var stdout bytes.Buffer

	if showOutput {
		emit, flushOutput := r.outputEmitter(stepID)
		ow := newOutputWriter(emit)
		cmd.Stdout = io.MultiWriter(&stdout, ow)
		cmd.Stderr = sl.Writer()
		err := cmd.Run()
		ow.Flush()
		flushOutput()
		return stdout.String(), err
	}

	cmd.Stdout = &stdout
	cmd.Stderr = sl.Writer()
	err := cmd.Run()
	return stdout.String(), err
}

func (r *Runner) execNoCapture(cmdStr string, sl *logging.StepLogger, showOutput bool, stepID string) error {
	cmd := exec.Command("sh", "-c", cmdStr)
	cmd.Env = r.buildEnv()

	if showOutput {
		emit, flushOutput := r.outputEmitter(stepID)
		ow := newOutputWriter(emit)
		cmd.Stdout = io.MultiWriter(sl.Writer(), ow)
		cmd.Stderr = sl.Writer()
		err := cmd.Run()
		ow.Flush()
		flushOutput()
		return err
	}

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
