package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/destis/pipe/internal/config"
	"github.com/destis/pipe/internal/logging"
	"github.com/destis/pipe/internal/parser"
	"github.com/destis/pipe/internal/runner"
	"github.com/destis/pipe/internal/state"
)

// version is set by goreleaser via ldflags
var version = "dev"

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: pipe <command|pipeline> [args]")
		fmt.Fprintln(os.Stderr, "commands: init, list, validate")
		os.Exit(1)
	}

	if os.Args[1] == "--version" || os.Args[1] == "-v" {
		fmt.Printf("pipe-%s\n", version)
		return
	}

	switch os.Args[1] {
	case "init":
		cmdInit()
	case "list":
		cmdList()
	case "validate":
		cmdValidate()
	default:
		runPipeline()
	}
}

func cmdInit() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: pipe init <name>")
		os.Exit(1)
	}
	name := os.Args[2]

	if err := os.MkdirAll(config.FilesDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	path := filepath.Join(config.FilesDir, name+".yaml")
	if _, err := os.Stat(path); err == nil {
		fmt.Fprintf(os.Stderr, "error: pipeline %q already exists at %s\n", name, path)
		os.Exit(1)
	}

	template := fmt.Sprintf(`name: %s
description: ""
steps:
  - id: hello
    run: "echo Hello from %s"
`, name, name)

	if err := os.WriteFile(path, []byte(template), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(path)
}

func cmdList() {
	infos, err := parser.ListPipelines()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if len(infos) == 0 {
		fmt.Println("no pipelines found — use 'pipe init <name>' to create one")
		return
	}

	// find max name width for alignment
	maxName := len("NAME")
	for _, info := range infos {
		if len(info.Name) > maxName {
			maxName = len(info.Name)
		}
	}

	fmt.Printf("%-*s  %s\n", maxName, "NAME", "DESCRIPTION")
	for _, info := range infos {
		fmt.Printf("%-*s  %s\n", maxName, info.Name, info.Description)
	}
}

func cmdValidate() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: pipe validate <name>")
		os.Exit(1)
	}
	name := os.Args[2]

	if err := parser.ValidatePipeline(name); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("pipeline %q is valid\n", name)
}

func runPipeline() {
	pipelineName := os.Args[1]
	var resumeID string

	for i := 2; i < len(os.Args); i++ {
		if os.Args[i] == "--resume" && i+1 < len(os.Args) {
			resumeID = os.Args[i+1]
			i++
		}
	}

	pipeline, err := parser.LoadPipeline(pipelineName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if err := config.EnsureDirs(pipeline.Name); err != nil {
		fmt.Fprintf(os.Stderr, "error creating dirs: %v\n", err)
		os.Exit(1)
	}

	var rs *state.RunState
	if resumeID != "" {
		rs, err = state.Load(pipeline.Name, resumeID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error loading state: %v\n", err)
			os.Exit(1)
		}
		rs.Status = "running"
	} else {
		rs = state.NewRunState(pipeline.Name)
	}

	log, err := logging.New(pipeline.Name, rs.RunID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating logger: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = log.Close() }()

	if resumeID != "" {
		log.Info("resuming pipeline %q (run %s)", pipeline.Name, rs.RunID)
	} else {
		log.Info("starting pipeline %q (run %s)", pipeline.Name, rs.RunID)
	}

	if err := state.Save(rs); err != nil {
		fmt.Fprintf(os.Stderr, "error saving initial state: %v\n", err)
		os.Exit(1)
	}

	r := runner.New(pipeline, rs, log)
	if resumeID != "" {
		r.RestoreEnvFromState()
	}

	if err := r.Run(); err != nil {
		os.Exit(1)
	}
}
