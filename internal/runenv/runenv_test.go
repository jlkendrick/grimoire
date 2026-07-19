package runenv

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"sync"
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
	env := New(dc, nil, Config{Present: func(v any) error {
		prints = append(prints, v)
		return nil
	}})

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

// TestSpellChecker_ValidatesFrontHalf exercises the whole load-time front
// half against a real cache: parse scroll → reconcile (extraction) →
// FromRitual → ValidatePipeline with the cache-backed existence check.
// No python needed — nothing runs; extraction is tree-sitter only.
func TestSpellChecker_ValidatesFrontHalf(t *testing.T) {
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

	check := SpellChecker(dc, nil)
	if err := check("fetch"); err != nil {
		t.Errorf("known spell rejected: %v", err)
	}
	if err := check("ghost"); err == nil {
		t.Error("unknown spell accepted")
	}

	valid := ir.FromRitual(&scroll.Ritual{Command: "report", Steps: []scroll.Step{
		{Id: "check", Spell: "fetch"},
		{Print: "check.n"},
	}})
	if err := graph.ValidatePipeline(valid, check); err != nil {
		t.Errorf("front half rejected a valid ritual: %v", err)
	}

	bad := ir.FromRitual(&scroll.Ritual{Command: "report", Steps: []scroll.Step{
		{Spell: "ghost"},
	}})
	if err := graph.ValidatePipeline(bad, check); err == nil {
		t.Error("front half accepted a ritual naming an unknown spell")
	}
}

// TestRealPipeline_GraphModeFromYAML is the full graph-mode path: mode:
// parsed from scroll.yaml, reconciled (validated by building), lowered,
// built, and run — two independent python spells fanning out from the
// seed and a third joining their outputs by reference.
func TestRealPipeline_GraphModeFromYAML(t *testing.T) {
	if !testsupport.PythonAvailable() {
		t.Skip("python3 not on PATH")
	}
	testsupport.SetupGrimoireHome(t)
	dir := testsupport.WithScrollDir(t)

	testsupport.WriteFile(t, filepath.Join(dir, "spells.py"), `def left(n: int = 1):
    return {"v": n * 2}


def right(n: int = 1):
    return {"v": n * 3}


def join(a: int = 0, b: int = 0):
    return a + b
`)
	scrollPath := testsupport.WriteScrollYAML(t, dir, `spells:
  - command: left
    path: spells.py
    function: left
  - command: right
    path: spells.py
    function: right
  - command: join
    path: spells.py
    function: join
rituals:
  - command: fan
    mode: graph
    steps:
      - id: l
        spell: left
      - id: r
        spell: right
      - id: joined
        spell: join
        params:
          a: l.v
          b: r.v
      - print: joined
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
	if got := dc.Pipelines["fan"].Mode; got != ir.ModeGraph {
		t.Fatalf("cached pipeline mode = %q, want graph", got)
	}

	var prints []any
	env := New(dc, nil, Config{Present: func(v any) error {
		prints = append(prints, v)
		return nil
	}})
	pipeline := dc.Pipelines["fan"]
	g, err := graph.BuildPipeline(&pipeline, env)
	if err != nil {
		t.Fatalf("BuildPipeline: %v", err)
	}
	if _, err := graph.Run(context.Background(), g, map[string]any{"n": 2}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// left(2) → 4, right(2) → 6, join(4, 6) → 10.
	if !slices.Equal(prints, []any{10.0}) {
		t.Errorf("prints = %v, want [10]", prints)
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
	env := New(dc, nil, Config{Present: func(v any) error {
		prints = append(prints, v)
		return nil
	}})

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

// TestObserver_SpellLifecycleEvents drives a two-step pipe ritual with a
// chatty first spell and asserts the event stream: per-spell ordering,
// distinct ids, decoded return values, the runtime version, and user
// prints arriving as stderr events instead of leaking to the terminal.
func TestObserver_SpellLifecycleEvents(t *testing.T) {
	if !testsupport.PythonAvailable() {
		t.Skip("python3 not on PATH")
	}
	testsupport.SetupGrimoireHome(t)
	dir := testsupport.WithScrollDir(t)

	testsupport.WriteFile(t, filepath.Join(dir, "spells.py"), `def loud(n: int = 1):
    print("working hard")
    return {"n": n + 1}


def quiet(n: int = 0):
    return n * 10
`)
	scrollPath := testsupport.WriteScrollYAML(t, dir, `spells:
  - command: loud
    path: spells.py
    function: loud
  - command: quiet
    path: spells.py
    function: quiet
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

	type event struct {
		kind, spell, line string
		id                int
		out               any
		rt                string
	}
	var mu sync.Mutex
	var events []event
	record := func(e event) {
		mu.Lock()
		events = append(events, e)
		mu.Unlock()
	}
	obs := &Observer{
		OnSpellStart: func(id int, spell string) {
			record(event{kind: "start", id: id, spell: spell})
		},
		OnSpellStderr: func(id int, line string) {
			record(event{kind: "stderr", id: id, line: line})
		},
		OnSpellFinish: func(id int, spell string, res FinishInfo) {
			if res.Err != nil {
				t.Errorf("finish err for %s: %v", spell, res.Err)
			}
			record(event{kind: "finish", id: id, spell: spell, out: res.Out, rt: res.RuntimeVersion})
		},
	}

	env := New(dc, nil, Config{Present: func(any) error { return nil }, Observer: obs})
	p := ir.FromRitual(&scroll.Ritual{Command: "chain", Steps: []scroll.Step{
		{Spell: "loud"},
		{Spell: "quiet"},
	}})
	g, err := graph.BuildPipeline(p, env)
	if err != nil {
		t.Fatalf("BuildPipeline: %v", err)
	}
	if _, err := graph.Run(context.Background(), g, map[string]any{"n": 1}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Pipe mode is deterministic: loud fully finishes before quiet starts.
	kinds := make([]string, len(events))
	for i, e := range events {
		kinds[i] = fmt.Sprintf("%s:%d", e.kind, e.id)
	}
	want := []string{"start:1", "stderr:1", "finish:1", "start:2", "finish:2"}
	if !slices.Equal(kinds, want) {
		t.Fatalf("event stream = %v, want %v", kinds, want)
	}
	if events[0].spell != "loud" || events[3].spell != "quiet" {
		t.Errorf("spell names = %q, %q", events[0].spell, events[3].spell)
	}
	if events[1].line != "working hard" {
		t.Errorf("stderr line = %q, want the user print", events[1].line)
	}
	if m, ok := events[2].out.(map[string]any); !ok || m["n"] != 2.0 {
		t.Errorf("loud finish out = %v, want decoded map with n=2", events[2].out)
	}
	if events[4].out != 20.0 {
		t.Errorf("quiet finish out = %v, want 20", events[4].out)
	}
	if !strings.Contains(events[2].rt, "python") {
		t.Errorf("runtime version = %q, want a python version string", events[2].rt)
	}
}
