package resolve_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	cache "github.com/jlkendrick/grimoire/internal/cache"
	resolve "github.com/jlkendrick/grimoire/internal/resolve"
	scroll "github.com/jlkendrick/grimoire/internal/scroll"
	ts "github.com/jlkendrick/grimoire/internal/testsupport"
)

const pyGreet = `def greet(name: str = "world"):
    return f"hello {name}"
`

const pyShout = `def shout(name: str):
    return name.upper()
`

const goSimple = `package sample

func DoubleIt(n int) int {
	return n * 2
}
`

// fixture writes scroll.yaml + the named source files into a fresh temp scroll
// dir under a configured GRIMOIRE_HOME, parses the scroll, and returns
// (parsedScroll, descriptorCache, scrollAbsPath). It captures stdout during
// the parse to suppress reconciler chatter.
func fixture(t *testing.T, scrollBody string, sources map[string]string) (*scroll.Scroll, *cache.DescriptorCache, string) {
	t.Helper()
	ts.SetupGrimoireHome(t)
	dir := ts.WithScrollDir(t)
	for name, body := range sources {
		ts.WriteFile(t, filepath.Join(dir, name), body)
	}
	scrollPath := ts.WriteScrollYAML(t, dir, scrollBody)

	s, err := scroll.ParseScroll(scrollPath)
	if err != nil {
		t.Fatalf("ParseScroll: %v", err)
	}
	dc, err := cache.ReadDescriptorCache(scrollPath)
	if err != nil {
		t.Fatalf("ReadDescriptorCache: %v", err)
	}
	return s, dc, scrollPath
}

// silentReconcile runs ReconcileScrollAndDescriptors with stdout redirected
// to /dev/null so reconciler progress prints don't pollute test output.
func silentReconcile(t *testing.T, s *scroll.Scroll, dc *cache.DescriptorCache) error {
	t.Helper()
	orig := os.Stdout
	devnull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("open devnull: %v", err)
	}
	os.Stdout = devnull
	defer func() {
		os.Stdout = orig
		_ = devnull.Close()
	}()
	return resolve.ReconcileScrollAndDescriptors(s, dc)
}

func TestReconcile_FreshScrollPopulatesCache(t *testing.T) {
	s, dc, _ := fixture(t,
		"spells:\n  - command: greet\n    path: greet.py\n    function: greet\n",
		map[string]string{"greet.py": pyGreet},
	)

	if err := silentReconcile(t, s, dc); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	got, ok := dc.Functions["greet"]
	if !ok {
		t.Fatalf("Functions['greet'] missing after reconcile; map=%+v", dc.Functions)
	}
	if got.FunctionName != "greet" {
		t.Errorf("FunctionName = %q, want greet", got.FunctionName)
	}
	if got.SpellHash == "" {
		t.Errorf("SpellHash empty; reconciler should have set it")
	}
	if dc.ScrollHash == "" {
		t.Errorf("ScrollHash empty; reconciler should have refreshed it")
	}
	if len(got.Params) == 0 {
		t.Errorf("expected params extracted from source, got 0")
	}
}

// TestReconcile_UnchangedScrollShortCircuits: a second reconcile on an
// already-reconciled scroll must not modify the cache file. Verifies the
// L19-21 short-circuit (scroll hash equality).
func TestReconcile_UnchangedScrollShortCircuits(t *testing.T) {
	s, dc, scrollPath := fixture(t,
		"spells:\n  - command: greet\n    path: greet.py\n    function: greet\n",
		map[string]string{"greet.py": pyGreet},
	)
	if err := silentReconcile(t, s, dc); err != nil {
		t.Fatalf("Reconcile 1: %v", err)
	}

	// Locate the on-disk cache file via the same scheme the cache package uses.
	cacheDir := filepath.Join(os.Getenv("GRIMOIRE_HOME"), "cache")
	entries, err := os.ReadDir(cacheDir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("expected 1 cache file, got %v err=%v", entries, err)
	}
	cacheFile := filepath.Join(cacheDir, entries[0].Name())
	stat1, err := os.Stat(cacheFile)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	// Sleep just long enough that any rewrite would change mtime.
	time.Sleep(20 * time.Millisecond)

	// Reload the scroll fresh (memo cleared) and reconcile again. Use the
	// same dc pointer so the in-memory ScrollHash from run 1 is preserved.
	scroll.ResetScrollCache()
	s2, err := scroll.ParseScroll(scrollPath)
	if err != nil {
		t.Fatalf("ParseScroll 2: %v", err)
	}
	if err := silentReconcile(t, s2, dc); err != nil {
		t.Fatalf("Reconcile 2: %v", err)
	}
	stat2, err := os.Stat(cacheFile)
	if err != nil {
		t.Fatalf("Stat 2: %v", err)
	}
	if !stat2.ModTime().Equal(stat1.ModTime()) {
		t.Errorf("cache file mtime changed despite unchanged scroll: %v -> %v", stat1.ModTime(), stat2.ModTime())
	}
}

// TestReconcile_SpellHashChangeReExtracts: editing scroll.yaml (e.g. setting
// an explicit interpreter) changes Spell.Hash, which must trigger
// re-extraction and a new SpellHash on the cached descriptor.
func TestReconcile_SpellHashChangeReExtracts(t *testing.T) {
	s, dc, scrollPath := fixture(t,
		"spells:\n  - command: greet\n    path: greet.py\n    function: greet\n",
		map[string]string{"greet.py": pyGreet},
	)
	if err := silentReconcile(t, s, dc); err != nil {
		t.Fatalf("Reconcile 1: %v", err)
	}
	originalSpellHash := dc.Functions["greet"].SpellHash
	if originalSpellHash == "" {
		t.Fatalf("SpellHash empty after fresh reconcile")
	}

	// Edit the scroll: add an interpreter so the spell's JSON shape changes.
	ts.WriteFile(t, scrollPath,
		"spells:\n  - command: greet\n    path: greet.py\n    function: greet\n    interpreter: /usr/bin/python3\n",
	)
	scroll.ResetScrollCache()
	s2, err := scroll.ParseScroll(scrollPath)
	if err != nil {
		t.Fatalf("ParseScroll: %v", err)
	}
	if err := silentReconcile(t, s2, dc); err != nil {
		t.Fatalf("Reconcile 2: %v", err)
	}
	updated := dc.Functions["greet"]
	if updated.SpellHash == originalSpellHash {
		t.Errorf("SpellHash unchanged after edit: %q", updated.SpellHash)
	}
	if updated.Interpreter != "/usr/bin/python3" {
		t.Errorf("Interpreter not propagated by merger: %q", updated.Interpreter)
	}
}

// TestReconcile_SourceFileChangeDoesNotReExtract: source-file changes are
// out-of-scope for the reconciler — staleness against source content is
// handled separately in cmd/root.go's run path. As long as the spell's
// scroll-side definition is unchanged, the reconciler should leave the
// descriptor's SpellHash alone (and skip re-extraction).
func TestReconcile_SourceFileChangeDoesNotReExtract(t *testing.T) {
	s, dc, _ := fixture(t,
		"spells:\n  - command: greet\n    path: greet.py\n    function: greet\n",
		map[string]string{"greet.py": pyGreet},
	)
	if err := silentReconcile(t, s, dc); err != nil {
		t.Fatalf("Reconcile 1: %v", err)
	}
	originalSpellHash := dc.Functions["greet"].SpellHash

	// Mutate source file body. Spell entry in scroll.yaml is unchanged.
	ts.WriteFile(t, filepath.Join(filepath.Dir(s.Path), "greet.py"),
		`def greet(name: str = "world"):
    return f"howdy {name}"
`)

	if err := silentReconcile(t, s, dc); err != nil {
		t.Fatalf("Reconcile 2: %v", err)
	}
	if dc.Functions["greet"].SpellHash != originalSpellHash {
		t.Errorf("SpellHash changed despite scroll being unchanged: %q -> %q",
			originalSpellHash, dc.Functions["greet"].SpellHash)
	}
}

func TestReconcile_RemovedSpellPruned(t *testing.T) {
	s, dc, scrollPath := fixture(t,
		"spells:\n  - command: greet\n    path: greet.py\n    function: greet\n  - command: shout\n    path: shout.py\n    function: shout\n",
		map[string]string{"greet.py": pyGreet, "shout.py": pyShout},
	)
	if err := silentReconcile(t, s, dc); err != nil {
		t.Fatalf("Reconcile 1: %v", err)
	}
	if _, ok := dc.Functions["shout"]; !ok {
		t.Fatalf("setup: shout not populated")
	}

	// Drop shout from the scroll.
	ts.WriteFile(t, scrollPath,
		"spells:\n  - command: greet\n    path: greet.py\n    function: greet\n",
	)
	scroll.ResetScrollCache()
	s2, err := scroll.ParseScroll(scrollPath)
	if err != nil {
		t.Fatalf("ParseScroll: %v", err)
	}
	if err := silentReconcile(t, s2, dc); err != nil {
		t.Fatalf("Reconcile 2: %v", err)
	}
	if _, ok := dc.Functions["shout"]; ok {
		t.Errorf("shout not pruned from cache")
	}
	if _, ok := dc.Functions["greet"]; !ok {
		t.Errorf("greet should still be present")
	}
}

func TestReconcile_RitualWithUnknownSpellErrors(t *testing.T) {
	s, dc, _ := fixture(t,
		`spells:
  - command: greet
    path: greet.py
    function: greet
rituals:
  - command: pipe
    steps:
      - spell: greet
      - spell: nonexistent
`,
		map[string]string{"greet.py": pyGreet},
	)
	err := silentReconcile(t, s, dc)
	if err == nil {
		t.Fatal("expected error for ritual referencing unknown spell, got nil")
	}
}

func TestReconcile_RitualAddedAndPruned(t *testing.T) {
	s, dc, scrollPath := fixture(t,
		`spells:
  - command: greet
    path: greet.py
    function: greet
rituals:
  - command: pipe
    steps:
      - spell: greet
`,
		map[string]string{"greet.py": pyGreet},
	)
	if err := silentReconcile(t, s, dc); err != nil {
		t.Fatalf("Reconcile 1: %v", err)
	}
	pd, ok := dc.Pipelines["pipe"]
	if !ok {
		t.Fatalf("pipeline 'pipe' not added; map=%+v", dc.Pipelines)
	}
	if len(pd.Steps) != 1 || pd.Steps[0].SpellName != "greet" {
		t.Errorf("Steps wrong: %+v", pd.Steps)
	}
	if pd.RitualHash == "" {
		t.Errorf("RitualHash empty")
	}

	// Drop ritual.
	ts.WriteFile(t, scrollPath,
		"spells:\n  - command: greet\n    path: greet.py\n    function: greet\n",
	)
	scroll.ResetScrollCache()
	s2, err := scroll.ParseScroll(scrollPath)
	if err != nil {
		t.Fatalf("ParseScroll: %v", err)
	}
	if err := silentReconcile(t, s2, dc); err != nil {
		t.Fatalf("Reconcile 2: %v", err)
	}
	if _, ok := dc.Pipelines["pipe"]; ok {
		t.Errorf("ritual not pruned from cache")
	}
}

func TestReconcile_RitualWithIfStep_Valid(t *testing.T) {
	s, dc, _ := fixture(t,
		`spells:
  - command: greet
    path: greet.py
    function: greet
  - command: shout
    path: shout.py
    function: shout
rituals:
  - command: pipe
    steps:
      - id: g
        spell: greet
      - if: g == "hello world"
        then:
          - spell: shout
`,
		map[string]string{"greet.py": pyGreet, "shout.py": pyShout},
	)
	if err := silentReconcile(t, s, dc); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	pd, ok := dc.Pipelines["pipe"]
	if !ok {
		t.Fatalf("pipeline 'pipe' not added; map=%+v", dc.Pipelines)
	}
	if len(pd.Steps) != 2 {
		t.Fatalf("expected 2 top-level steps, got %d", len(pd.Steps))
	}
	if pd.Steps[1].Kind() != "if" {
		t.Errorf("expected step 2 to be an if-step, got %s", pd.Steps[1].Kind())
	}
	if pd.Steps[1].Condition != `g == "hello world"` {
		t.Errorf("Condition = %q", pd.Steps[1].Condition)
	}
	if len(pd.Steps[1].Then) != 1 || pd.Steps[1].Then[0].SpellName != "shout" {
		t.Errorf("Then branch wrong: %+v", pd.Steps[1].Then)
	}
}

func TestReconcile_RitualUnknownSpellInBranch(t *testing.T) {
	s, dc, _ := fixture(t,
		`spells:
  - command: greet
    path: greet.py
    function: greet
rituals:
  - command: pipe
    steps:
      - id: g
        spell: greet
      - if: g == "x"
        then:
          - spell: nonexistent
`,
		map[string]string{"greet.py": pyGreet},
	)
	if err := silentReconcile(t, s, dc); err == nil {
		t.Fatal("expected error for unknown spell in branch, got nil")
	}
}

func TestReconcile_RitualBadConditionSyntax(t *testing.T) {
	s, dc, _ := fixture(t,
		`spells:
  - command: greet
    path: greet.py
    function: greet
rituals:
  - command: pipe
    steps:
      - id: g
        spell: greet
      - if: g >
        then:
          - spell: greet
`,
		map[string]string{"greet.py": pyGreet},
	)
	if err := silentReconcile(t, s, dc); err == nil {
		t.Fatal("expected error for malformed condition, got nil")
	}
}

func TestReconcile_RitualFirstStepIsIfErrors(t *testing.T) {
	s, dc, _ := fixture(t,
		`spells:
  - command: greet
    path: greet.py
    function: greet
rituals:
  - command: pipe
    steps:
      - if: "true"
        then:
          - spell: greet
`,
		map[string]string{"greet.py": pyGreet},
	)
	if err := silentReconcile(t, s, dc); err == nil {
		t.Fatal("expected error for if-step as first step, got nil")
	}
}

func TestReconcile_RitualConditionRefOutOfScopeErrors(t *testing.T) {
	s, dc, _ := fixture(t,
		`spells:
  - command: greet
    path: greet.py
    function: greet
rituals:
  - command: pipe
    steps:
      - spell: greet
      - if: missing.x
        then:
          - spell: greet
`,
		map[string]string{"greet.py": pyGreet},
	)
	if err := silentReconcile(t, s, dc); err == nil {
		t.Fatal("expected error for condition referencing out-of-scope id, got nil")
	}
}

func TestReconcile_RitualParamRefToBranchInternalIdErrors(t *testing.T) {
	// An id declared inside `then` must not be visible to siblings of the
	// if-step. The reconciler rejects the second top-level step's
	// `inner.field` accessor as out-of-scope. (A bare `inner` with no
	// accessor would be treated as a literal string, not a reference —
	// consistent with how getStepReference detects refs.)
	s, dc, _ := fixture(t,
		`spells:
  - command: greet
    path: greet.py
    function: greet
  - command: shout
    path: shout.py
    function: shout
rituals:
  - command: pipe
    steps:
      - spell: greet
      - if: "true"
        then:
          - id: inner
            spell: greet
      - spell: shout
        params:
          name: inner.field
`,
		map[string]string{"greet.py": pyGreet, "shout.py": pyShout},
	)
	if err := silentReconcile(t, s, dc); err == nil {
		t.Fatal("expected error for sibling reference to branch-internal id, got nil")
	}
}

func TestReconcile_RitualIfStepWithIdErrors(t *testing.T) {
	s, dc, _ := fixture(t,
		`spells:
  - command: greet
    path: greet.py
    function: greet
rituals:
  - command: pipe
    steps:
      - spell: greet
      - id: bogus
        if: "true"
        then:
          - spell: greet
`,
		map[string]string{"greet.py": pyGreet},
	)
	if err := silentReconcile(t, s, dc); err == nil {
		t.Fatal("expected error for if-step with id, got nil")
	}
}

func TestReconcile_RitualEmptyThenErrors(t *testing.T) {
	s, dc, _ := fixture(t,
		`spells:
  - command: greet
    path: greet.py
    function: greet
rituals:
  - command: pipe
    steps:
      - spell: greet
      - if: "true"
        then: []
`,
		map[string]string{"greet.py": pyGreet},
	)
	if err := silentReconcile(t, s, dc); err == nil {
		t.Fatal("expected error for empty then branch, got nil")
	}
}

func TestReconcile_RitualWithLetStep_Valid(t *testing.T) {
	s, dc, _ := fixture(t,
		`spells:
  - command: greet
    path: greet.py
    function: greet
  - command: shout
    path: shout.py
    function: shout
rituals:
  - command: pipe
    steps:
      - id: g
        spell: greet
      - let: is_hello
        value: g == "hello world"
      - if: is_hello
        then:
          - spell: shout
`,
		map[string]string{"greet.py": pyGreet, "shout.py": pyShout},
	)
	if err := silentReconcile(t, s, dc); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	pd, ok := dc.Pipelines["pipe"]
	if !ok {
		t.Fatalf("pipeline 'pipe' not added")
	}
	if len(pd.Steps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(pd.Steps))
	}
	if pd.Steps[1].Kind() != "let" {
		t.Errorf("step 2 kind = %q, want let", pd.Steps[1].Kind())
	}
	if pd.Steps[1].Let != "is_hello" {
		t.Errorf("Let = %q", pd.Steps[1].Let)
	}
	if pd.Steps[1].Value != `g == "hello world"` {
		t.Errorf("Value = %q", pd.Steps[1].Value)
	}
}

func TestReconcile_RitualLetStepMissingValueErrors(t *testing.T) {
	s, dc, _ := fixture(t,
		`spells:
  - command: greet
    path: greet.py
    function: greet
rituals:
  - command: pipe
    steps:
      - spell: greet
      - let: x
`,
		map[string]string{"greet.py": pyGreet},
	)
	if err := silentReconcile(t, s, dc); err == nil {
		t.Fatal("expected error for let-step without value, got nil")
	}
}

func TestReconcile_RitualLetStepBadValueErrors(t *testing.T) {
	s, dc, _ := fixture(t,
		`spells:
  - command: greet
    path: greet.py
    function: greet
rituals:
  - command: pipe
    steps:
      - spell: greet
      - let: x
        value: "1 + "
`,
		map[string]string{"greet.py": pyGreet},
	)
	if err := silentReconcile(t, s, dc); err == nil {
		t.Fatal("expected error for malformed let value, got nil")
	}
}

func TestReconcile_RitualLetStepOutOfScopeRefErrors(t *testing.T) {
	s, dc, _ := fixture(t,
		`spells:
  - command: greet
    path: greet.py
    function: greet
rituals:
  - command: pipe
    steps:
      - spell: greet
      - let: x
        value: missing.field
`,
		map[string]string{"greet.py": pyGreet},
	)
	if err := silentReconcile(t, s, dc); err == nil {
		t.Fatal("expected error for let value referencing out-of-scope id, got nil")
	}
}

func TestReconcile_RitualLetStepAsFirstErrors(t *testing.T) {
	s, dc, _ := fixture(t,
		`spells:
  - command: greet
    path: greet.py
    function: greet
rituals:
  - command: pipe
    steps:
      - let: x
        value: "true"
      - spell: greet
`,
		map[string]string{"greet.py": pyGreet},
	)
	if err := silentReconcile(t, s, dc); err == nil {
		t.Fatal("expected error for let-step as first step, got nil")
	}
}

func TestReconcile_RitualLetStepReboundReplacesValue(t *testing.T) {
	// Same name written twice — both pass static validation; runtime
	// re-binds. Verifies that re-assignment isn't rejected at reconcile
	// time (no const-ness contract today).
	s, dc, _ := fixture(t,
		`spells:
  - command: greet
    path: greet.py
    function: greet
rituals:
  - command: pipe
    steps:
      - id: g
        spell: greet
      - let: x
        value: g
      - let: x
        value: g
`,
		map[string]string{"greet.py": pyGreet},
	)
	if err := silentReconcile(t, s, dc); err != nil {
		t.Errorf("expected re-binding the same let name to be allowed: %v", err)
	}
}

// TestReconcile_GoAndPythonMix exercises the language-dispatch path: one .py
// spell and one .go spell in the same scroll both extract correctly.
func TestReconcile_GoAndPythonMix(t *testing.T) {
	s, dc, _ := fixture(t,
		"spells:\n  - command: greet\n    path: greet.py\n    function: greet\n  - command: doubleit\n    path: math.go\n    function: DoubleIt\n",
		map[string]string{"greet.py": pyGreet, "math.go": goSimple},
	)
	if err := silentReconcile(t, s, dc); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if _, ok := dc.Functions["greet"]; !ok {
		t.Errorf("python spell missing")
	}
	got, ok := dc.Functions["doubleit"]
	if !ok {
		t.Fatalf("go spell missing")
	}
	if got.FunctionName != "DoubleIt" {
		t.Errorf("FunctionName = %q, want DoubleIt", got.FunctionName)
	}
	if len(got.Params) == 0 {
		t.Errorf("expected at least 1 param extracted from Go func, got 0")
	}
}
