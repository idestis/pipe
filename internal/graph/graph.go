package graph

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/getpipe-dev/pipe/internal/model"
)

// Graph represents a DAG of pipeline step dependencies.
type Graph struct {
	Deps       map[string][]string // predecessors: step → steps it depends on
	Dependents map[string][]string // successors: step → steps that depend on it
	InDegree   map[string]int      // number of predecessors
	Order      []string            // step IDs preserving YAML order
	Warnings   []string            // non-fatal issues (e.g. unknown dep refs)
}

// pipeVarPattern matches $PIPE_<NAME> and ${PIPE_<NAME>} references in shell commands.
var pipeVarPattern = regexp.MustCompile(`\$\{?PIPE_([A-Z0-9_]+)\}?`)

// envKey mirrors runner.EnvKey: joins parts with _, replaces hyphens, uppercases.
func envKey(parts ...string) string {
	joined := strings.Join(parts, "_")
	joined = strings.ReplaceAll(joined, "-", "_")
	return "PIPE_" + strings.ToUpper(joined)
}

// Build constructs a dependency graph from pipeline steps.
// It adds explicit edges from depends_on and implicit edges from $PIPE_* variable references.
// Returns an error for cycles, unknown step refs, or self-dependencies.
func Build(steps []model.Step) (*Graph, error) {
	g := &Graph{
		Deps:       make(map[string][]string),
		Dependents: make(map[string][]string),
		InDegree:   make(map[string]int),
	}

	// Build lookup maps
	stepByID := make(map[string]model.Step)
	envToStep := make(map[string]string) // PIPE_<KEY> → step ID that produces it
	for _, s := range steps {
		g.Order = append(g.Order, s.ID)
		g.InDegree[s.ID] = 0
		stepByID[s.ID] = s

		// Map env keys to producing step
		key := envKey(s.ID)
		envToStep[key] = s.ID
		if s.Run.IsSubRuns() {
			for _, sr := range s.Run.SubRuns {
				subKey := envKey(s.ID, sr.ID)
				envToStep[subKey] = s.ID
			}
		}
	}

	// Track edges to avoid duplicates
	edgeSet := make(map[string]bool)
	addEdge := func(from, to string) {
		key := from + " -> " + to
		if edgeSet[key] {
			return
		}
		edgeSet[key] = true
		g.Deps[to] = append(g.Deps[to], from)
		g.Dependents[from] = append(g.Dependents[from], to)
		g.InDegree[to]++
	}

	for _, s := range steps {
		// Explicit depends_on edges
		for _, dep := range s.DependsOn.Steps {
			if dep == s.ID {
				return nil, fmt.Errorf("step %q: self-dependency", s.ID)
			}
			if _, ok := stepByID[dep]; !ok {
				g.Warnings = append(g.Warnings, fmt.Sprintf("step %q: unknown dependency %q (ignored)", s.ID, dep))
				continue
			}
			addEdge(dep, s.ID)
		}

		// Implicit edges from $PIPE_* variable references
		for _, ref := range findPipeRefs(s) {
			if producer, ok := envToStep[ref]; ok && producer != s.ID {
				addEdge(producer, s.ID)
			}
		}
	}

	// Cycle detection using Kahn's algorithm
	if err := detectCycle(g); err != nil {
		return nil, err
	}

	return g, nil
}

// findPipeRefs extracts all PIPE_* variable names referenced in a step's run commands.
func findPipeRefs(s model.Step) []string {
	var refs []string
	seen := make(map[string]bool)

	collect := func(cmd string) {
		matches := pipeVarPattern.FindAllStringSubmatch(cmd, -1)
		for _, m := range matches {
			// m[0] is full match like $PIPE_FOO or ${PIPE_FOO}
			// Reconstruct the full env key name
			varName := "PIPE_" + m[1]
			if !seen[varName] {
				seen[varName] = true
				refs = append(refs, varName)
			}
		}
	}

	if s.Run.IsSingle() {
		collect(s.Run.Single)
	}
	for _, cmd := range s.Run.Strings {
		collect(cmd)
	}
	for _, sr := range s.Run.SubRuns {
		collect(sr.Run)
	}

	return refs
}

// detectCycle uses Kahn's algorithm to detect cycles.
func detectCycle(g *Graph) error {
	inDeg := make(map[string]int)
	for id, d := range g.InDegree {
		inDeg[id] = d
	}

	var queue []string
	for _, id := range g.Order {
		if inDeg[id] == 0 {
			queue = append(queue, id)
		}
	}

	processed := 0
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		processed++
		for _, dep := range g.Dependents[curr] {
			inDeg[dep]--
			if inDeg[dep] == 0 {
				queue = append(queue, dep)
			}
		}
	}

	if processed < len(g.Order) {
		// Find steps involved in cycle for better error message
		var inCycle []string
		for _, id := range g.Order {
			if inDeg[id] > 0 {
				inCycle = append(inCycle, id)
			}
		}
		return fmt.Errorf("dependency cycle detected among steps: %s", strings.Join(inCycle, ", "))
	}

	return nil
}
