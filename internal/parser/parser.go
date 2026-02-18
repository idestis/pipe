package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/idestis/pipe/internal/config"
	"github.com/idestis/pipe/internal/hub"
	"github.com/idestis/pipe/internal/model"
	"github.com/idestis/pipe/internal/resolve"
	"gopkg.in/yaml.v3"
)

// PipelineInfo holds lightweight metadata about a pipeline.
type PipelineInfo struct {
	Name        string
	Description string
	Source      string // "local" or "hub"
	Alias       string // alias pointing to this pipe, if any
	Version     string // active tag for hub pipes
}

func LoadPipeline(name string) (*model.Pipeline, error) {
	path := filepath.Join(config.FilesDir, name+".yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading pipeline %q: %w", name, err)
	}

	var p model.Pipeline
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parsing pipeline %q: %w", name, err)
	}

	if p.Name == "" {
		p.Name = name
	}

	if err := Validate(&p); err != nil {
		return nil, fmt.Errorf("validating pipeline %q: %w", name, err)
	}
	return &p, nil
}

// Validate checks a pipeline for structural errors such as missing or
// duplicate step IDs and missing run fields.
func Validate(p *model.Pipeline) error {
	for key := range p.Vars {
		if !validVarKey(key) {
			return fmt.Errorf("invalid var key %q — use only letters, digits, hyphens, and underscores", key)
		}
	}

	ids := make(map[string]bool)
	for i, s := range p.Steps {
		if s.ID == "" {
			return fmt.Errorf("step %d: missing id", i)
		}
		if ids[s.ID] {
			return fmt.Errorf("step %d: duplicate id %q", i, s.ID)
		}
		ids[s.ID] = true

		if !s.Run.IsSingle() && !s.Run.IsStrings() && !s.Run.IsSubRuns() {
			return fmt.Errorf("step %q: missing run field", s.ID)
		}
	}
	return nil
}

// Warnings returns non-fatal warnings about the pipeline configuration.
func Warnings(p *model.Pipeline) []string {
	var warns []string

	// Collect env var names produced by sensitive steps
	sensitiveVars := make(map[string]string) // env var → step ID
	for _, s := range p.Steps {
		if !s.Sensitive {
			continue
		}
		sensitiveVars[envKey(s.ID)] = s.ID
		if s.Run.IsSubRuns() {
			for _, sr := range s.Run.SubRuns {
				if sr.Sensitive {
					sensitiveVars[envKey(s.ID, sr.ID)] = s.ID + "/" + sr.ID
				}
			}
		}
	}

	for _, s := range p.Steps {
		// Warn: cached + sensitive means step won't re-execute and env var won't be available
		if s.Cached.Enabled && s.Sensitive {
			warns = append(warns, fmt.Sprintf(
				"step %q: cached + sensitive — step will be skipped on cache hit, output is not stored and $%s will not be set",
				s.ID, envKey(s.ID),
			))
		}

		// Warn: referencing a sensitive step's env var (it will be re-executed on resume)
		for varName, srcID := range sensitiveVars {
			if referencesVar(s, varName) {
				warns = append(warns, fmt.Sprintf(
					"step %q: references $%s from sensitive step %q — that step will always re-execute on resume",
					s.ID, varName, srcID,
				))
			}
		}
	}
	return warns
}

// validVarKey checks that a variable key contains only letters, digits, hyphens,
// and underscores, and is non-empty.
func validVarKey(key string) bool {
	if len(key) == 0 {
		return false
	}
	for _, c := range key {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		case c == '-' || c == '_':
		default:
			return false
		}
	}
	return true
}

// envKey mirrors runner.EnvKey: joins parts with _, replaces hyphens, uppercases.
func envKey(parts ...string) string {
	joined := strings.Join(parts, "_")
	joined = strings.ReplaceAll(joined, "-", "_")
	return "PIPE_" + strings.ToUpper(joined)
}

// referencesVar checks if any run command in a step contains the given variable name.
func referencesVar(s model.Step, varName string) bool {
	check := func(cmd string) bool {
		return strings.Contains(cmd, "$"+varName) || strings.Contains(cmd, "${"+varName+"}")
	}
	if s.Run.IsSingle() && check(s.Run.Single) {
		return true
	}
	for _, cmd := range s.Run.Strings {
		if check(cmd) {
			return true
		}
	}
	for _, sr := range s.Run.SubRuns {
		if check(sr.Run) {
			return true
		}
	}
	return false
}

// ValidatePipeline loads and validates a pipeline by name.
// It returns nil on success or a descriptive error.
func ValidatePipeline(name string) error {
	_, err := LoadPipeline(name)
	return err
}

// ListPipelines reads all *.yaml files from config.FilesDir and returns
// lightweight info (name + description) sorted by name.
func ListPipelines() ([]PipelineInfo, error) {
	pattern := filepath.Join(config.FilesDir, "*.yaml")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("listing pipelines: %w", err)
	}

	var infos []PipelineInfo
	for _, path := range matches {
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: skipping %s: %v\n", filepath.Base(path), err)
			continue
		}
		var p model.Pipeline
		if err := yaml.Unmarshal(data, &p); err != nil {
			fmt.Fprintf(os.Stderr, "warning: skipping %s: %v\n", filepath.Base(path), err)
			continue
		}
		name := p.Name
		if name == "" {
			base := filepath.Base(path)
			name = strings.TrimSuffix(base, ".yaml")
		}
		infos = append(infos, PipelineInfo{Name: name, Description: p.Description})
	}

	sort.Slice(infos, func(i, j int) bool {
		return infos[i].Name < infos[j].Name
	})
	return infos, nil
}

// LoadPipelineFromPath loads a pipeline from an explicit file path.
func LoadPipelineFromPath(path, displayName string) (*model.Pipeline, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading pipeline %q: %w", displayName, err)
	}

	var p model.Pipeline
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parsing pipeline %q: %w", displayName, err)
	}

	if p.Name == "" {
		p.Name = displayName
	}

	if err := Validate(&p); err != nil {
		return nil, fmt.Errorf("validating pipeline %q: %w", displayName, err)
	}
	return &p, nil
}

// ListAllPipelines merges local files and hub pipes into a unified list.
func ListAllPipelines() ([]PipelineInfo, error) {
	// Load aliases for reverse lookup
	aliases, err := resolve.LoadAliases()
	if err != nil {
		return nil, err
	}
	aliasMap := make(map[string]string) // target → alias name
	for name, entry := range aliases {
		aliasMap[entry.Target] = name
	}

	var infos []PipelineInfo

	// Local pipes
	localPipes, err := ListPipelines()
	if err != nil {
		return nil, err
	}
	for _, lp := range localPipes {
		info := PipelineInfo{
			Name:        lp.Name,
			Description: lp.Description,
			Source:      "local",
		}
		if a, ok := aliasMap[lp.Name]; ok {
			info.Alias = a
		}
		infos = append(infos, info)
	}

	// Hub pipes
	hubPipes, err := hub.ListPipes()
	if err != nil {
		return nil, err
	}
	for _, hp := range hubPipes {
		fullName := hp.Owner + "/" + hp.Name
		path := hub.ContentPath(hp.Owner, hp.Name, hp.ActiveTag)
		desc := ""
		if data, err := os.ReadFile(path); err == nil {
			var p model.Pipeline
			if err := yaml.Unmarshal(data, &p); err == nil {
				desc = p.Description
			}
		}
		info := PipelineInfo{
			Name:        fullName,
			Description: desc,
			Source:      "hub",
			Version:     hp.ActiveTag,
		}
		if a, ok := aliasMap[fullName]; ok {
			info.Alias = a
		}
		infos = append(infos, info)
	}

	sort.Slice(infos, func(i, j int) bool {
		return infos[i].Name < infos[j].Name
	})
	return infos, nil
}
