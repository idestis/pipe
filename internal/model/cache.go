package model

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// CacheField supports two YAML forms:
//   - bool:    cache: true → {Enabled: true, ExpireAfter: ""}
//   - mapping: cache: {expireAfter: "1h"} → {Enabled: true, ExpireAfter: "1h"}
type CacheField struct {
	Enabled     bool
	ExpireAfter string
}

func (c *CacheField) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		var b bool
		if err := value.Decode(&b); err != nil {
			return fmt.Errorf("cache: expected true/false, got %q", value.Value)
		}
		c.Enabled = b
		return nil

	case yaml.MappingNode:
		var m struct {
			ExpireAfter string `yaml:"expireAfter"`
		}
		if err := value.Decode(&m); err != nil {
			return fmt.Errorf("cache: decoding mapping: %w", err)
		}
		c.Enabled = true
		c.ExpireAfter = m.ExpireAfter
		return nil

	default:
		return fmt.Errorf("cache: must be a bool or a mapping with expireAfter")
	}
}
