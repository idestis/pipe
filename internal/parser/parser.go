package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/destis/pipe/internal/config"
	"github.com/destis/pipe/internal/model"
	"gopkg.in/yaml.v3"
)

// PipelineInfo holds lightweight metadata about a pipeline.
type PipelineInfo struct {
	Name        string
	Description string
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
			continue
		}
		var p model.Pipeline
		if err := yaml.Unmarshal(data, &p); err != nil {
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
