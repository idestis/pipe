package model

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

type Pipeline struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Steps       []Step `yaml:"steps"`
}

type Step struct {
	ID        string   `yaml:"id"`
	Run       RunField `yaml:"run"`
	Parallel  bool     `yaml:"parallel"`
	Sensitive bool     `yaml:"sensitive"`
	Retry     int      `yaml:"retry"`
}

// RunField supports three YAML forms:
//   - scalar string: single command
//   - sequence of strings: parallel plain commands (no output capture)
//   - sequence of mappings: parallel named sub-runs (output captured per sub-run)
type RunField struct {
	Single  string
	Strings []string
	SubRuns []SubRun
}

type SubRun struct {
	ID        string `yaml:"id"`
	Run       string `yaml:"run"`
	Sensitive bool   `yaml:"sensitive"`
}

func (r *RunField) IsSingle() bool   { return r.Single != "" }
func (r *RunField) IsStrings() bool  { return len(r.Strings) > 0 }
func (r *RunField) IsSubRuns() bool  { return len(r.SubRuns) > 0 }

func (r *RunField) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		r.Single = value.Value
		return nil

	case yaml.SequenceNode:
		if len(value.Content) == 0 {
			return fmt.Errorf("run: empty sequence")
		}
		first := value.Content[0]
		switch first.Kind {
		case yaml.ScalarNode:
			var strs []string
			if err := value.Decode(&strs); err != nil {
				return fmt.Errorf("run: decoding string list: %w", err)
			}
			r.Strings = strs
			return nil
		case yaml.MappingNode:
			var subs []SubRun
			if err := value.Decode(&subs); err != nil {
				return fmt.Errorf("run: decoding sub-run list: %w", err)
			}
			r.SubRuns = subs
			return nil
		default:
			return fmt.Errorf("run: unexpected sequence element kind %d", first.Kind)
		}

	default:
		return fmt.Errorf("run: unexpected YAML kind %d", value.Kind)
	}
}
