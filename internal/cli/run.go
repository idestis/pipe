package cli

import (
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/idestis/pipe/internal/config"
	"github.com/idestis/pipe/internal/hub"
	"github.com/idestis/pipe/internal/logging"
	"github.com/idestis/pipe/internal/parser"
	"github.com/idestis/pipe/internal/resolve"
	"github.com/idestis/pipe/internal/runner"
	"github.com/idestis/pipe/internal/state"
)

func showPipelineHelp(name string) error {
	ref, err := resolve.Resolve(name)
	if err != nil {
		return err
	}

	pipeline, err := parser.LoadPipelineFromPath(ref.Path, ref.Name)
	if err != nil {
		return err
	}

	fmt.Printf("Pipeline: %s\n", ref.Name)
	if pipeline.Description != "" {
		fmt.Printf("          %s\n", pipeline.Description)
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

	// For hub pipes, check for local modifications
	if ref.Kind == resolve.KindHub {
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

	for _, w := range parser.Warnings(pipeline) {
		log.Warn(w)
	}

	if err := config.EnsureDirs(pipeline.Name); err != nil {
		return fmt.Errorf("%s", friendlyError(err))
	}

	var rs *state.RunState
	if resumeFlag != "" {
		rs, err = state.Load(pipeline.Name, resumeFlag)
		if err != nil {
			return err
		}
		rs.Status = "running"
	} else {
		rs = state.NewRunState(pipeline.Name)
	}

	plog, err := logging.New(pipeline.Name, rs.RunID)
	if err != nil {
		return fmt.Errorf("%s", friendlyError(err))
	}
	defer func() { _ = plog.Close() }()

	if resumeFlag != "" {
		plog.Log("resuming pipeline %q (run %s)", pipeline.Name, rs.RunID)
	} else {
		plog.Log("starting pipeline %q (run %s)", pipeline.Name, rs.RunID)
	}

	if err := state.Save(rs); err != nil {
		return fmt.Errorf("%s", friendlyError(err))
	}

	vars := runner.ResolveVars(pipeline.Vars, overrides)
	r := runner.New(pipeline, rs, plog, vars)
	if resumeFlag != "" {
		r.RestoreEnvFromState()
	}

	return r.Run()
}
