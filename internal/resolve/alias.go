package resolve

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/idestis/pipe/internal/config"
)

// AliasEntry maps an alias name to a target pipe reference.
type AliasEntry struct {
	Target    string    `json:"target"`
	CreatedAt time.Time `json:"created_at"`
}

// AliasFile is the on-disk format for aliases.json.
type AliasFile struct {
	Aliases map[string]AliasEntry `json:"aliases"`
}

// LoadAliases reads aliases.json. Returns an empty map if the file doesn't exist.
func LoadAliases() (map[string]AliasEntry, error) {
	data, err := os.ReadFile(config.AliasesPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return make(map[string]AliasEntry), nil
		}
		return nil, fmt.Errorf("reading aliases: %w", err)
	}
	var f AliasFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parsing aliases: %w", err)
	}
	if f.Aliases == nil {
		f.Aliases = make(map[string]AliasEntry)
	}
	return f.Aliases, nil
}

// SaveAliases writes aliases.json atomically.
func SaveAliases(aliases map[string]AliasEntry) error {
	f := AliasFile{Aliases: aliases}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling aliases: %w", err)
	}
	tmp := config.AliasesPath + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("writing aliases: %w", err)
	}
	return os.Rename(tmp, config.AliasesPath)
}

// GetAlias returns the target for an alias, or empty string if not found.
func GetAlias(name string) (string, error) {
	aliases, err := LoadAliases()
	if err != nil {
		return "", err
	}
	entry, ok := aliases[name]
	if !ok {
		return "", nil
	}
	return entry.Target, nil
}

// SetAlias creates or updates an alias.
func SetAlias(name, target string) error {
	aliases, err := LoadAliases()
	if err != nil {
		return err
	}
	aliases[name] = AliasEntry{
		Target:    target,
		CreatedAt: time.Now(),
	}
	return SaveAliases(aliases)
}

// DeleteAlias removes an alias. Returns an error if the alias doesn't exist.
func DeleteAlias(name string) error {
	aliases, err := LoadAliases()
	if err != nil {
		return err
	}
	if _, ok := aliases[name]; !ok {
		return fmt.Errorf("alias %q not found", name)
	}
	delete(aliases, name)
	return SaveAliases(aliases)
}

// ReassignAlias changes the target of an existing alias.
func ReassignAlias(name, newTarget string) error {
	aliases, err := LoadAliases()
	if err != nil {
		return err
	}
	if _, ok := aliases[name]; !ok {
		return fmt.Errorf("alias %q not found", name)
	}
	aliases[name] = AliasEntry{
		Target:    newTarget,
		CreatedAt: time.Now(),
	}
	return SaveAliases(aliases)
}

// AliasInfo holds display information for an alias.
type AliasInfo struct {
	Name   string
	Target string
}

// ListAliases returns all aliases sorted by name.
func ListAliases() ([]AliasInfo, error) {
	aliases, err := LoadAliases()
	if err != nil {
		return nil, err
	}
	var list []AliasInfo
	for name, entry := range aliases {
		list = append(list, AliasInfo{Name: name, Target: entry.Target})
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Name < list[j].Name
	})
	return list, nil
}

// FindAliasForTarget returns the alias name that points to the given target, or "".
func FindAliasForTarget(target string) (string, error) {
	aliases, err := LoadAliases()
	if err != nil {
		return "", err
	}
	for name, entry := range aliases {
		if entry.Target == target {
			return name, nil
		}
	}
	return "", nil
}
