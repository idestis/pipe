package resolve

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/getpipe-dev/pipe/internal/config"
	"github.com/getpipe-dev/pipe/internal/hub"
)

// PipeKind distinguishes local pipes from hub pipes.
type PipeKind int

const (
	KindLocal PipeKind = iota
	KindHub
)

// PipeRef is the resolved reference to a pipe on disk.
type PipeRef struct {
	Kind    PipeKind
	Name    string // display name (e.g., "demo" or "idestis/demo")
	Path    string // absolute path to YAML file
	Owner   string // hub owner (empty for local)
	Pipe    string // hub pipe name (empty for local)
	Tag     string // active tag (empty for local)
	Alias   string // alias that resolved to this pipe (empty if none)
}

// ParsePipeArg parses an argument like "owner/name:tag" into owner, name, tag.
// If no tag is given, tag defaults to "".
// If the argument has no slash, owner is "" and name is the full arg.
func ParsePipeArg(arg string) (owner, name, tag string) {
	// Split off tag
	if i := strings.LastIndex(arg, ":"); i >= 0 {
		tag = arg[i+1:]
		arg = arg[:i]
	}
	// Split owner/name
	if i := strings.Index(arg, "/"); i >= 0 {
		owner = arg[:i]
		name = arg[i+1:]
	} else {
		name = arg
	}
	log.Debug("ParsePipeArg", "owner", owner, "name", name, "tag", tag)
	return
}

// Resolve performs the 3-step lookup: alias → hub → local.
func Resolve(input string) (*PipeRef, error) {
	owner, name, tag := ParsePipeArg(input)
	alias := ""
	log.Debug("resolving pipe", "input", input, "owner", owner, "name", name, "tag", tag)

	// Step 1: Check aliases (only for plain names without owner)
	if owner == "" {
		log.Debug("checking alias", "name", name)
		target, err := GetAlias(name)
		if err != nil {
			return nil, err
		}
		if target != "" {
			alias = name
			owner, name, _ = ParsePipeArg(target)
			log.Debug("alias resolved", "alias", alias, "target", target, "owner", owner, "name", name)
			// Keep original tag if user specified one; otherwise use resolved tag
			// (ParsePipeArg on target might return a tag from alias, but we prefer user's tag)
		}
	}

	// Step 2: Hub pipe (has owner)
	if owner != "" {
		log.Debug("checking hub index", "owner", owner, "name", name)
		idx, err := hub.LoadIndex(owner, name)
		if err != nil {
			return nil, err
		}
		if idx != nil {
			log.Debug("hub index found", "owner", owner, "name", name, "tags", len(idx.Tags), "activeTag", idx.ActiveTag)
			if tag == "" {
				// Read HEAD ref to determine what HEAD points to
				headRef, err := hub.ReadHeadRef(owner, name)
				if err == nil && headRef.Value != "" {
					if headRef.Kind == hub.HeadKindBlob {
						// HEAD points to a blob — use blob path directly
						blobPath := hub.BlobPath(owner, name, headRef.Value)
						if _, serr := os.Stat(blobPath); serr == nil {
							shortSHA := headRef.Value
						if len(shortSHA) > 12 {
							shortSHA = shortSHA[:12]
						}
						log.Debug("resolved to hub blob", "owner", owner, "name", name, "sha256", shortSHA)
							return &PipeRef{
								Kind:  KindHub,
								Name:  owner + "/" + name,
								Path:  blobPath,
								Owner: owner,
								Pipe:  name,
								Tag:   headRef.Value,
								Alias: alias,
							}, nil
						}
					}
					tag = headRef.Value
				} else {
					tag = idx.ActiveTag
				}
			}
			if tag == "" {
				tag = "latest"
			}
			path := hub.ContentPath(owner, name, tag)
			if _, err := os.Stat(path); err == nil {
				log.Debug("resolved to hub tag", "owner", owner, "name", name, "tag", tag, "path", path)
				return &PipeRef{
					Kind:  KindHub,
					Name:  owner + "/" + name,
					Path:  path,
					Owner: owner,
					Pipe:  name,
					Tag:   tag,
					Alias: alias,
				}, nil
			}
			return nil, fmt.Errorf("tag %q not pulled for %s/%s\n  run \"pipe pull %s/%s:%s\" first", tag, owner, name, owner, name, tag)
		}
		// No index — this hub pipe hasn't been pulled
		return nil, fmt.Errorf("pipe %q not found\n  run \"pipe pull %s/%s\" to get it from Pipe Hub, or \"pipe list\" to see local pipes", owner+"/"+name, owner, name)
	}

	// Step 3: Local pipe
	log.Debug("checking local pipe", "name", name)
	path := filepath.Join(config.FilesDir, name+".yaml")
	if _, err := os.Stat(path); err == nil {
		log.Debug("resolved to local pipe", "name", name, "path", path)
		return &PipeRef{
			Kind: KindLocal,
			Name: name,
			Path: path,
		}, nil
	}

	return nil, fmt.Errorf("pipeline %q not found\n  run \"pipe list\" to see available pipelines, or \"pipe init %s\" to create one", name, name)
}
