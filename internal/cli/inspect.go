package cli

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/idestis/pipe/internal/graph"
	"github.com/idestis/pipe/internal/hub"
	"github.com/idestis/pipe/internal/parser"
	"github.com/idestis/pipe/internal/resolve"
	"github.com/spf13/cobra"
)

var inspectCmd = &cobra.Command{
	Use:   "inspect <name>",
	Short: "Show detailed info about a pipeline",
	Args:  exactArgs(1, "pipe inspect <name>"),
	RunE: func(cmd *cobra.Command, args []string) error {
		ref, err := resolve.Resolve(args[0])
		if err != nil {
			return err
		}
		log.Debug("resolved pipeline", "name", ref.Name, "kind", ref.Kind, "path", ref.Path)

		pipeline, err := parser.LoadPipelineFromPath(ref.Path, ref.Name)
		if err != nil {
			return fmt.Errorf("loading pipeline: %w", err)
		}
		log.Debug("loaded pipeline", "steps", len(pipeline.Steps), "vars", len(pipeline.Vars))

		// Pre-load hub data so all debug logs fire before user-facing output
		var headRef *hub.HeadRef
		var idx *hub.Index
		if ref.Kind == resolve.KindHub {
			headRef, _ = hub.ReadHeadRef(ref.Owner, ref.Pipe)
			log.Debug("HEAD ref", "owner", ref.Owner, "pipe", ref.Pipe, "ref", headRef)
			idx, _ = hub.LoadIndex(ref.Owner, ref.Pipe)
			if idx != nil {
				log.Debug("index loaded", "tags", len(idx.Tags), "activeTag", idx.ActiveTag)
			}
		}

		// --- user-facing output below ---

		fmt.Printf("Name:        %s\n", ref.Name)
		if ref.Alias != "" {
			fmt.Printf("Alias:       %s\n", ref.Alias)
		}
		fmt.Printf("Source:      %s\n", kindStr(ref.Kind))
		fmt.Printf("Path:        %s\n", ref.Path)

		if pipeline.Description != "" {
			fmt.Printf("Description: %s\n", pipeline.Description)
		}
		fmt.Printf("Steps:       %d\n", len(pipeline.Steps))

		// Build graph for dependency info
		g, _ := graph.Build(pipeline.Steps)
		for _, step := range pipeline.Steps {
			deps := ""
			if g != nil && len(g.Deps[step.ID]) > 0 {
				deps = fmt.Sprintf("  (depends on: %s)", strings.Join(g.Deps[step.ID], ", "))
			}
			fmt.Printf("  - %s%s\n", step.ID, deps)
		}

		fmt.Printf("Vars:        %d\n", len(pipeline.Vars))

		if ref.Kind == resolve.KindHub {
			if headRef != nil {
				if headRef.Kind == hub.HeadKindBlob {
					fmt.Printf("HEAD:        sha256:%s (detached)\n", short(headRef.Value, 12))
				} else {
					fmt.Printf("HEAD:        %s\n", headRef.Value)
				}
			}
			fmt.Printf("Active Tag:  %s\n", ref.Tag)

			if idx != nil && len(idx.Tags) > 0 {
				fmt.Println("\nPulled Tags:")
				for tag, rec := range idx.Tags {
					active := ""
					if tag == idx.ActiveTag {
						active = " (active)"
					}

					// Tag type
					tagType := "symlink"
					if rec.Editable {
						tagType = "editable"
					}

					// Dirty check
					dirtyMarker := ""
					dirty, derr := hub.IsDirty(ref.Owner, ref.Pipe, tag)
					if derr == nil && dirty {
						dirtyMarker = " [dirty]"
					}

					pulledAt := ""
					if !rec.PulledAt.IsZero() {
						pulledAt = fmt.Sprintf("  pulled=%s", rec.PulledAt.Format("2006-01-02 15:04"))
					}
					createdAt := ""
					if !rec.CreatedAt.IsZero() {
						createdAt = fmt.Sprintf("  created=%s", rec.CreatedAt.Format("2006-01-02 15:04"))
					}

					fmt.Printf("  %-16s [%s] sha256=%s%s%s%s%s\n",
						tag, tagType, short(rec.SHA256, 12), pulledAt, createdAt, active, dirtyMarker)
				}
			}
		}

		return nil
	},
}

func kindStr(k resolve.PipeKind) string {
	switch k {
	case resolve.KindLocal:
		return "local"
	case resolve.KindHub:
		return "hub"
	default:
		return "unknown"
	}
}
