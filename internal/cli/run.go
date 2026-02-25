package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/charmbracelet/log"
	"github.com/getpipe-dev/pipe/internal/config"
	"github.com/getpipe-dev/pipe/internal/hub"
	"github.com/getpipe-dev/pipe/internal/logging"
	"github.com/getpipe-dev/pipe/internal/parser"
	"github.com/getpipe-dev/pipe/internal/resolve"
	"github.com/getpipe-dev/pipe/internal/runner"
	"github.com/getpipe-dev/pipe/internal/state"
	"github.com/getpipe-dev/pipe/internal/ui"
)

func showPipelineHelp(name string) error {
	ref, err := resolve.Resolve(name)
	if err != nil {
		return err
	}
	log.Debug("resolved pipeline for help", "name", ref.Name, "kind", ref.Kind, "path", ref.Path)

	pipeline, err := parser.LoadPipelineFromPath(ref.Path, ref.Name)
	if err != nil {
		return err
	}

	fmt.Printf("Pipeline: %s\n", ref.Name)
	if pipeline.Description != "" {
		fmt.Printf("          %s\n", pipeline.Description)
	}
	if pipeline.DotFile != "" {
		fmt.Printf("Env File: %s\n", pipeline.DotFile)
	}
	fmt.Println()

	// Usage line
	fmt.Printf("Usage:\n  pipe %s", ref.Name)
	if len(pipeline.Vars) > 0 {
		fmt.Print(" [-- KEY=value ...]")
	}
	fmt.Println()
	fmt.Printf("  pipe %s --resume <run-id>\n", ref.Name)
	fmt.Println()

	// Show available vars
	if len(pipeline.Vars) > 0 {
		fmt.Println("Variables:")
		maxKey := 0
		for k := range pipeline.Vars {
			if len(k) > maxKey {
				maxKey = len(k)
			}
		}
		for k, v := range pipeline.Vars {
			fmt.Printf("  %-*s  (default: %q)\n", maxKey, k, v)
		}
		fmt.Println()
	}

	// Steps summary
	fmt.Printf("Steps: %d\n", len(pipeline.Steps))
	for _, s := range pipeline.Steps {
		fmt.Printf("  - %s\n", s.ID)
	}

	return nil
}

func runPipeline(name string, overrides map[string]string) error {
	ref, err := resolve.Resolve(name)
	if err != nil {
		return err
	}
	log.Debug("resolved pipeline", "name", ref.Name, "kind", ref.Kind, "path", ref.Path, "tag", ref.Tag)

	// For hub pipes, check for local modifications
	if ref.Kind == resolve.KindHub {
		log.Debug("checking hub pipe integrity", "owner", ref.Owner, "name", ref.Pipe, "tag", ref.Tag)
		dirty, err := hub.IsDirty(ref.Owner, ref.Pipe, ref.Tag)
		if err != nil {
			log.Warn("could not check integrity", "err", err)
		} else if dirty {
			editable, _ := hub.IsTagEditable(ref.Owner, ref.Pipe, ref.Tag)
			if editable {
				log.Warn("editable tag has unpushed changes — running with local content", "pipe", ref.Name, "tag", ref.Tag)
			} else {
				log.Warn("local modifications detected — running with uncommitted changes", "pipe", ref.Name, "tag", ref.Tag)
			}
		}
	}

	pipeline, err := parser.LoadPipelineFromPath(ref.Path, ref.Name)
	if err != nil {
		if isYAMLError(err) {
			return fmt.Errorf("invalid YAML in pipeline %q: %v", ref.Name, unwrapYAMLError(err))
		}
		return err
	}
	log.Debug("parsed pipeline", "name", pipeline.Name, "steps", len(pipeline.Steps), "vars", len(pipeline.Vars))

	for _, w := range parser.Warnings(pipeline) {
		log.Warn(w)
	}

	if err := config.EnsureDirs(pipeline.Name); err != nil {
		return fmt.Errorf("%s", friendlyError(err))
	}

	var rs *state.RunState
	if resumeFlag != "" {
		log.Debug("resuming run", "runID", resumeFlag)
		rs, err = state.Load(pipeline.Name, resumeFlag)
		if err != nil {
			return err
		}
		rs.Status = "running"
		log.Debug("loaded run state", "runID", rs.RunID, "status", rs.Status)
	} else {
		rs = state.NewRunState(pipeline.Name)
		log.Debug("new run state", "runID", rs.RunID)
	}

	var statusUI *ui.StatusUI
	var plog *logging.Logger
	if verbosity == 0 && ui.IsTTY(os.Stderr) {
		log.SetLevel(log.WarnLevel)
		plog, err = logging.New(pipeline.Name, rs.RunID, logging.FileOnly())
		statusUI = ui.NewStatusUI(os.Stderr, pipeline.Steps)
	} else {
		plog, err = logging.New(pipeline.Name, rs.RunID)
	}
	if err != nil {
		return fmt.Errorf("%s", friendlyError(err))
	}
	defer func() { _ = plog.Close() }()

	if err := logging.RotateLogs(pipeline.Name); err != nil {
		log.Warn("log rotation failed", "err", err)
	}

	if resumeFlag != "" {
		plog.Log("resuming pipeline %q (run %s)", pipeline.Name, rs.RunID)
	} else {
		plog.Log("starting pipeline %q (run %s)", pipeline.Name, rs.RunID)
	}

	if err := state.Save(rs); err != nil {
		return fmt.Errorf("%s", friendlyError(err))
	}

	if resumeFlag == "" {
		if err := state.RotateStates(pipeline.Name, rs.RunID); err != nil {
			log.Warn("state rotation failed", "err", err)
		}
	}

	var dotFileVars map[string]string
	if pipeline.DotFile != "" {
		var dotFileWarns []string
		dotFileVars, dotFileWarns, err = runner.ParseDotFile(pipeline.DotFile)
		switch {
		case errors.Is(err, os.ErrNotExist):
			// Missing file: silent skip.
		case err != nil:
			log.Warn("dot_file could not be fully read", "path", pipeline.DotFile, "err", err)
		}
		for _, w := range dotFileWarns {
			log.Warn(w)
		}
	}

	vars, resolveWarns := runner.ResolveVars(pipeline.Vars, dotFileVars, overrides)
	for _, w := range resolveWarns {
		log.Warn(w)
	}
	for _, w := range runner.UnmatchedEnvVarWarnings(pipeline.Vars) {
		log.Warn(w)
	}
	log.Debug("resolved variables", "total", len(vars), "overrides", len(overrides))
	r := runner.New(pipeline, rs, plog, vars, statusUI, verbosity)
	if resumeFlag != "" {
		r.RestoreEnvFromState()
	}

	return r.Run()
}
