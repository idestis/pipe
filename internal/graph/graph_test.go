package graph

import (
	"strings"
	"testing"

	"github.com/idestis/pipe/internal/model"
)

func steps(defs ...stepDef) []model.Step {
	var out []model.Step
	for _, d := range defs {
		s := model.Step{
			ID:  d.id,
			Run: d.run,
		}
		if len(d.deps) > 0 {
			s.DependsOn = model.DependsOnField{Steps: d.deps}
		}
		out = append(out, s)
	}
	return out
}

type stepDef struct {
	id   string
	run  model.RunField
	deps []string
}

func single(cmd string) model.RunField {
	return model.RunField{Single: cmd}
}

func TestBuild_LinearChain(t *testing.T) {
	ss := steps(
		stepDef{id: "a", run: single("echo a")},
		stepDef{id: "b", run: single("echo b"), deps: []string{"a"}},
		stepDef{id: "c", run: single("echo c"), deps: []string{"b"}},
	)
	g, err := Build(ss)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if g.InDegree["a"] != 0 {
		t.Fatalf("expected a in-degree 0, got %d", g.InDegree["a"])
	}
	if g.InDegree["b"] != 1 {
		t.Fatalf("expected b in-degree 1, got %d", g.InDegree["b"])
	}
	if g.InDegree["c"] != 1 {
		t.Fatalf("expected c in-degree 1, got %d", g.InDegree["c"])
	}
}

func TestBuild_Diamond(t *testing.T) {
	ss := steps(
		stepDef{id: "a", run: single("echo a")},
		stepDef{id: "b", run: single("echo b"), deps: []string{"a"}},
		stepDef{id: "c", run: single("echo c"), deps: []string{"a"}},
		stepDef{id: "d", run: single("echo d"), deps: []string{"b", "c"}},
	)
	g, err := Build(ss)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if g.InDegree["a"] != 0 {
		t.Fatalf("expected a in-degree 0, got %d", g.InDegree["a"])
	}
	if g.InDegree["d"] != 2 {
		t.Fatalf("expected d in-degree 2, got %d", g.InDegree["d"])
	}
}

func TestBuild_FullyIndependent(t *testing.T) {
	ss := steps(
		stepDef{id: "a", run: single("echo a")},
		stepDef{id: "b", run: single("echo b")},
		stepDef{id: "c", run: single("echo c")},
	)
	g, err := Build(ss)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, id := range []string{"a", "b", "c"} {
		if g.InDegree[id] != 0 {
			t.Fatalf("expected %s in-degree 0, got %d", id, g.InDegree[id])
		}
	}
}

func TestBuild_ImplicitDeps(t *testing.T) {
	ss := steps(
		stepDef{id: "get-version", run: single("git describe --tags")},
		stepDef{id: "build", run: single("docker build -t app:$PIPE_GET_VERSION .")},
	)
	g, err := Build(ss)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if g.InDegree["build"] != 1 {
		t.Fatalf("expected build in-degree 1, got %d", g.InDegree["build"])
	}
	if len(g.Deps["build"]) != 1 || g.Deps["build"][0] != "get-version" {
		t.Fatalf("expected build to depend on get-version, got %v", g.Deps["build"])
	}
}

func TestBuild_ImplicitDepsBraces(t *testing.T) {
	ss := steps(
		stepDef{id: "get-version", run: single("git describe --tags")},
		stepDef{id: "build", run: single("docker build -t app:${PIPE_GET_VERSION} .")},
	)
	g, err := Build(ss)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if g.InDegree["build"] != 1 {
		t.Fatalf("expected build in-degree 1, got %d", g.InDegree["build"])
	}
}

func TestBuild_ImplicitDepsSubRun(t *testing.T) {
	ss := []model.Step{
		{ID: "fetch", Run: model.RunField{SubRuns: []model.SubRun{
			{ID: "token", Run: "vault read token"},
		}}},
		{ID: "deploy", Run: single("curl -H $PIPE_FETCH_TOKEN https://api")},
	}
	g, err := Build(ss)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if g.InDegree["deploy"] != 1 {
		t.Fatalf("expected deploy in-degree 1, got %d", g.InDegree["deploy"])
	}
}

func TestBuild_CycleDetection(t *testing.T) {
	ss := steps(
		stepDef{id: "a", run: single("echo a"), deps: []string{"b"}},
		stepDef{id: "b", run: single("echo b"), deps: []string{"a"}},
	)
	_, err := Build(ss)
	if err == nil {
		t.Fatal("expected cycle error")
	}
	if !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("expected error about cycle, got: %v", err)
	}
}

func TestBuild_UnknownStepRef(t *testing.T) {
	ss := steps(
		stepDef{id: "a", run: single("echo a"), deps: []string{"nonexistent"}},
	)
	_, err := Build(ss)
	if err == nil {
		t.Fatal("expected error for unknown step ref")
	}
	if !strings.Contains(err.Error(), "unknown dependency") {
		t.Fatalf("expected error about unknown dependency, got: %v", err)
	}
}

func TestBuild_SelfDep(t *testing.T) {
	ss := steps(
		stepDef{id: "a", run: single("echo a"), deps: []string{"a"}},
	)
	_, err := Build(ss)
	if err == nil {
		t.Fatal("expected error for self-dependency")
	}
	if !strings.Contains(err.Error(), "self-dependency") {
		t.Fatalf("expected error about self-dependency, got: %v", err)
	}
}

func TestBuild_MixedExplicitAndImplicit(t *testing.T) {
	ss := steps(
		stepDef{id: "get-version", run: single("git describe")},
		stepDef{id: "lint", run: single("go vet ./...")},
		stepDef{id: "build", run: single("go build -ldflags \"-X main.version=$PIPE_GET_VERSION\""), deps: []string{"lint"}},
	)
	g, err := Build(ss)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// build depends on lint (explicit) and get-version (implicit)
	if g.InDegree["build"] != 2 {
		t.Fatalf("expected build in-degree 2, got %d", g.InDegree["build"])
	}
}

func TestBuild_ImplicitDepsInStringsList(t *testing.T) {
	ss := []model.Step{
		{ID: "get-sha", Run: single("git rev-parse HEAD")},
		{ID: "build", Run: model.RunField{Strings: []string{
			"docker build -t app:$PIPE_GET_SHA .",
			"echo done",
		}}},
	}
	g, err := Build(ss)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if g.InDegree["build"] != 1 {
		t.Fatalf("expected build in-degree 1, got %d", g.InDegree["build"])
	}
}

func TestBuild_OrderPreserved(t *testing.T) {
	ss := steps(
		stepDef{id: "z", run: single("echo z")},
		stepDef{id: "a", run: single("echo a")},
		stepDef{id: "m", run: single("echo m")},
	)
	g, err := Build(ss)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(g.Order) != 3 || g.Order[0] != "z" || g.Order[1] != "a" || g.Order[2] != "m" {
		t.Fatalf("expected order [z a m], got %v", g.Order)
	}
}

func TestBuild_ThreeNodeCycle(t *testing.T) {
	ss := steps(
		stepDef{id: "a", run: single("echo a"), deps: []string{"c"}},
		stepDef{id: "b", run: single("echo b"), deps: []string{"a"}},
		stepDef{id: "c", run: single("echo c"), deps: []string{"b"}},
	)
	_, err := Build(ss)
	if err == nil {
		t.Fatal("expected cycle error")
	}
	if !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("expected error about cycle, got: %v", err)
	}
}

func TestBuild_NoDuplicateEdges(t *testing.T) {
	// Both explicit and implicit dep on same step should create single edge
	ss := steps(
		stepDef{id: "get-version", run: single("git describe")},
		stepDef{id: "build", run: single("echo $PIPE_GET_VERSION"), deps: []string{"get-version"}},
	)
	g, err := Build(ss)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if g.InDegree["build"] != 1 {
		t.Fatalf("expected build in-degree 1 (deduped), got %d", g.InDegree["build"])
	}
}

func TestBuild_VarRefIgnoresSelf(t *testing.T) {
	// A step referencing its own PIPE_ var should not create a self-edge
	ss := steps(
		stepDef{id: "build", run: single("echo $PIPE_BUILD")},
	)
	g, err := Build(ss)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if g.InDegree["build"] != 0 {
		t.Fatalf("expected build in-degree 0 (self ref ignored), got %d", g.InDegree["build"])
	}
}
