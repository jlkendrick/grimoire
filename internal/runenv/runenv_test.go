package runenv

import (
	"context"
	"path/filepath"
	"slices"
	"testing"

	cache "github.com/jlkendrick/grimoire/internal/cache"
	graph "github.com/jlkendrick/grimoire/internal/graph"
	ir "github.com/jlkendrick/grimoire/internal/ir"
	resolve "github.com/jlkendrick/grimoire/internal/resolve"
	scroll "github.com/jlkendrick/grimoire/internal/scroll"
	testsupport "github.com/jlkendrick/grimoire/internal/testsupport"
)

// TestRealPipeline_EndToEnd is the proof that the new core runs real code:
// real scroll.yaml, real tree-sitter extraction via the reconciler, real
// python subprocesses through runtime.Run — orchestrated entirely by
// graph.BuildPipeline + graph.Run.
func TestRealPipeline_EndToEnd(t *testing.T) {
	if !testsupport.PythonAvailable() {
		t.Skip("python3 not on PATH")
	}
	testsupport.SetupGrimoireHome(t)
	dir := testsupport.WithScrollDir(t)

	// Spells RETURN their output; the runtime's wrapper JSON-encodes the
	// return value onto stdout and reroutes any user prints to stderr.
	testsupport.WriteFile(t, filepath.Join(dir, "spells.py"), `def fetch(n: int = 2):
    return {"ok": True, "n": n}


def double(n: int = 0):
    return n * 2
`)
	scrollPath := testsupport.WriteScrollYAML(t, dir, `spells:
  - command: fetch
    path: spells.py
    function: fetch
  - command: double
    path: spells.py
    function: double
`)

	sc, err := scroll.ParseScroll(scrollPath)
	if err != nil {
		t.Fatalf("ParseScroll: %v", err)
	}
	dc, err := cache.ReadDescriptorCache(scrollPath)
	if err != nil {
		t.Fatalf("ReadDescriptorCache: %v", err)
	}
	if err := resolve.ReconcileScrollAndDescriptors(sc, dc); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	var prints []any
	env := New(dc, nil, func(v any) error {
		prints = append(prints, v)
		return nil
	})

	p := &ir.Pipeline{Command: "demo", Steps: []ir.Step{
		{Spell: "fetch", Id: "check"},
		{If: "check.ok && check.n > 1",
			Then: []ir.Step{
				{Spell: "double", Id: "doubled", Params: map[string]any{"n": "check.n"}},
				{Print: "doubled"},
			},
			Else: []ir.Step{{Print: `"skipped"`}},
		},
	}}

	g, err := graph.BuildPipeline(p, env)
	if err != nil {
		t.Fatalf("BuildPipeline: %v", err)
	}
	if _, err := graph.Run(context.Background(), g, map[string]any{"n": 3}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// fetch(n=3) → {"ok":true,"n":3}; condition true; double(n=3) → 6.
	if !slices.Equal(prints, []any{6.0}) {
		t.Errorf("prints = %v, want [6]", prints)
	}
}

// TestRealPipeline_ElseBranch flips the condition through the seed and
// checks the else-branch print.
func TestRealPipeline_ElseBranch(t *testing.T) {
	if !testsupport.PythonAvailable() {
		t.Skip("python3 not on PATH")
	}
	testsupport.SetupGrimoireHome(t)
	dir := testsupport.WithScrollDir(t)

	testsupport.WriteFile(t, filepath.Join(dir, "spells.py"), `def fetch(n: int = 2):
    return {"ok": True, "n": n}
`)
	scrollPath := testsupport.WriteScrollYAML(t, dir, `spells:
  - command: fetch
    path: spells.py
    function: fetch
`)

	sc, err := scroll.ParseScroll(scrollPath)
	if err != nil {
		t.Fatalf("ParseScroll: %v", err)
	}
	dc, err := cache.ReadDescriptorCache(scrollPath)
	if err != nil {
		t.Fatalf("ReadDescriptorCache: %v", err)
	}
	if err := resolve.ReconcileScrollAndDescriptors(sc, dc); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	var prints []any
	env := New(dc, nil, func(v any) error {
		prints = append(prints, v)
		return nil
	})

	p := &ir.Pipeline{Command: "demo", Steps: []ir.Step{
		{Spell: "fetch", Id: "check"},
		{If: "check.n > 1",
			Then: []ir.Step{{Print: "check.n"}},
			Else: []ir.Step{{Print: `"skipped"`}},
		},
	}}

	g, err := graph.BuildPipeline(p, env)
	if err != nil {
		t.Fatalf("BuildPipeline: %v", err)
	}
	if _, err := graph.Run(context.Background(), g, map[string]any{"n": 1}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if !slices.Equal(prints, []any{"skipped"}) {
		t.Errorf("prints = %v, want [skipped]", prints)
	}
}
