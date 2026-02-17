package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/charmbracelet/log"
	"github.com/idestis/pipe/internal/config"
	"github.com/idestis/pipe/internal/logging"
	"github.com/idestis/pipe/internal/parser"
	"github.com/idestis/pipe/internal/runner"
	"github.com/idestis/pipe/internal/state"
)

func runPipeline(name string) error {
	pipeline, err := parser.LoadPipeline(name)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("pipeline %q not found\n  run \"pipe list\" to see available pipelines, or \"pipe init %s\" to create one", name, name)
		}
		if errors.Is(err, os.ErrPermission) {
			return fmt.Errorf("permission denied reading pipeline %q — check file permissions", name)
		}
		if isYAMLError(err) {
			return fmt.Errorf("invalid YAML in pipeline %q: %v", name, unwrapYAMLError(err))
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

	r := runner.New(pipeline, rs, plog)
	if resumeFlag != "" {
		r.RestoreEnvFromState()
	}

	return r.Run()
}
