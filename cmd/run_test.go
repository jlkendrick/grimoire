package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	cache "github.com/jlkendrick/grimoire/internal/cache"
	desc "github.com/jlkendrick/grimoire/internal/descriptor"
	extract "github.com/jlkendrick/grimoire/internal/extract"
	resolve "github.com/jlkendrick/grimoire/internal/resolve"
	scroll "github.com/jlkendrick/grimoire/internal/scroll"
	utils "github.com/jlkendrick/grimoire/internal/utils"
)

// writeCacheDirect writes a DescriptorCache JSON file to the on-disk location
// that ReadDescriptorCache(scrollPath) would read, bypassing
// cache.AddFunctionDescriptor so the test can choose the descriptor state
// directly.
func writeCacheDirect(t *testing.T, scrollPath string, descriptors map[string]desc.FunctionDescriptor) {
	t.Helper()
	home, err := utils.GrimoireHome()
	if err != nil {
		t.Fatalf("GrimoireHome: %v", err)
	}
	pathHash, err := utils.HashFilePath(scrollPath)
	if err != nil {
		t.Fatalf("HashFilePath: %v", err)
	}
	scrollHash, err := utils.HashFile(scrollPath)
	if err != nil {
		t.Fatalf("HashFile scroll: %v", err)
	}
	cacheFile := filepath.Join(home, "cache", pathHash+".json")
	c := cache.DescriptorCache{
		Version:    cache.CACHE_VERSION,
		ScrollHash: scrollHash,
		ScrollPath: scrollPath,
		Functions:  descriptors,
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if err := os.WriteFile(cacheFile, b, 0644); err != nil {
		t.Fatalf("WriteFile cache: %v", err)
	}
	cache.ResetCache()
}

func readCacheDirect(t *testing.T, scrollPath string) cache.DescriptorCache {
	t.Helper()
	home, err := utils.GrimoireHome()
	if err != nil {
		t.Fatalf("GrimoireHome: %v", err)
	}
	pathHash, err := utils.HashFilePath(scrollPath)
	if err != nil {
		t.Fatalf("HashFilePath: %v", err)
	}
	cacheFile := filepath.Join(home, "cache", pathHash+".json")
	b, err := os.ReadFile(cacheFile)
	if err != nil {
		t.Fatalf("ReadFile cache: %v", err)
	}
	var c cache.DescriptorCache
	if err := json.Unmarshal(b, &c); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	return c
}

func isRequired(ann map[string][]string) bool {
	v, ok := ann[cobra.BashCompOneRequiredFlag]
	return ok && len(v) > 0 && v[0] == "true"
}

// TestGenerateCommands_FlagsMatchParams unit-tests cobra flag generation given
// a hand-built descriptor cache with explicit Kind values.
func TestGenerateCommands_FlagsMatchParams(t *testing.T) {
	c := &cache.DescriptorCache{
		Version:    cache.CACHE_VERSION,
		ScrollPath: "/tmp/fake/scroll.yaml",
		Functions: map[string]desc.FunctionDescriptor{
			"greet": {
				CommandName:         "greet",
				FunctionName:        "greet",
				AbsPathToSourceFile: "/tmp/fake/greet.py",
				RelPathToSourceFile: "greet.py",
				ScrollPath:          "/tmp/fake/scroll.yaml",
				Params: []desc.ParamDescriptor{
					{Name: "n", ResolvedType: &desc.TypeInfo{Kind: desc.TypeKindPrimitive, Name: "int"}, Default: "1"},
					{Name: "who", ResolvedType: &desc.TypeInfo{Kind: desc.TypeKindPrimitive, Name: "str"}, Default: nil},
				},
			},
		},
	}

	commands, err := GenerateCommands(c, nil, "", nil)
	if err != nil {
		t.Fatalf("GenerateCommands: %v", err)
	}
	if len(commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(commands))
	}
	cmd := commands[0]
	if cmd.Use != "greet" {
		t.Errorf("Use = %q, want %q", cmd.Use, "greet")
	}

	nFlag := cmd.Flags().Lookup("n")
	if nFlag == nil {
		t.Fatalf("--n flag missing")
	}
	if nFlag.Value.Type() != "int" {
		t.Errorf("--n type = %q, want int", nFlag.Value.Type())
	}
	if nFlag.DefValue != "1" {
		t.Errorf("--n DefValue = %q, want 1", nFlag.DefValue)
	}
	if isRequired(nFlag.Annotations) {
		t.Errorf("--n should not be required (has default)")
	}

	whoFlag := cmd.Flags().Lookup("who")
	if whoFlag == nil {
		t.Fatalf("--who flag missing")
	}
	if whoFlag.Value.Type() != "string" {
		t.Errorf("--who type = %q, want string", whoFlag.Value.Type())
	}
	if !isRequired(whoFlag.Annotations) {
		t.Errorf("--who should be required (no default)")
	}
}

// TestRun_OutputCorrect drives a generated dynamic command through the runtime
// using a hand-built descriptor cache (Kind set, Interpreter set, hashes
// consistent) so the run path doesn't trip over open issues in the extractor's
// Kind handling or in cache.AddFunctionDescriptor's hash field.
func TestRun_OutputCorrect(t *testing.T) {
	if !pythonAvailable() {
		t.Skip("python3 not on PATH")
	}
	setupTestEnv(t)
	dir := withScrollDir(t)

	scrollPath := filepath.Join(dir, "scroll.yaml")
	writeFile(t, scrollPath, "spells:\n  - command: greet\n    path: greet.py\n    function: greet\n")
	srcPath := filepath.Join(dir, "greet.py")
	writeFile(t, srcPath, `def greet(name: str = "world"):
    print(f"hello {name}")
`)

	srcHash, err := utils.HashFile(srcPath)
	if err != nil {
		t.Fatalf("HashFile: %v", err)
	}
	writeCacheDirect(t, scrollPath, map[string]desc.FunctionDescriptor{
		"greet": {
			CommandName:         "greet",
			FunctionName:        "greet",
			AbsPathToSourceFile: srcPath,
			RelPathToSourceFile: "greet.py",
			ScrollPath:          scrollPath,
			Interpreter:         "python3",
			SourceHash:          srcHash,
			Params: []desc.ParamDescriptor{
				{Name: "name", ResolvedType: &desc.TypeInfo{Kind: desc.TypeKindPrimitive, Name: "str"}, Default: "world"},
			},
		},
	})

	c, err := cache.ReadDescriptorCache(scrollPath)
	if err != nil {
		t.Fatalf("ReadDescriptorCache: %v", err)
	}
	commands, err := GenerateCommands(c, nil, "", nil)
	if err != nil {
		t.Fatalf("GenerateCommands: %v", err)
	}
	parent := &cobra.Command{Use: "test"}
	for _, cm := range commands {
		parent.AddCommand(cm)
	}

	stdout, _ := captureOutput(t, func() {
		parent.SetArgs([]string{"greet", "--name", "test"})
		if err := parent.Execute(); err != nil {
			t.Fatalf("parent.Execute: %v", err)
		}
	})
	if !strings.Contains(stdout, "hello test") {
		t.Errorf("expected stdout to contain 'hello test', got:\n%s", stdout)
	}
}

// TestStaleness_SourceFile verifies the source-file staleness primitive: when
// the source file's hash diverges from the cached descriptor's per-descriptor
// SourceHash, a re-extraction must produce a descriptor whose SourceHash
// equals the new file's hash.
func TestStaleness_SourceFile(t *testing.T) {
	setupTestEnv(t)
	dir := withScrollDir(t)
	resetRootCmdState(t)

	srcPath := filepath.Join(dir, "greet.py")
	writeFile(t, srcPath, `def greet(name: str):
    print(name)
`)

	captureOutput(t, func() {
		rootCmd.SetArgs([]string{"add", "greet.py:greet"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("add: %v", err)
		}
	})

	scrollPath := filepath.Join(dir, "scroll.yaml")
	cache.ResetCache()
	c1, err := cache.ReadDescriptorCache(scrollPath)
	if err != nil {
		t.Fatalf("ReadDescriptorCache: %v", err)
	}
	originalDesc := c1.Functions["greet"]
	originalSourceHash := originalDesc.SourceHash
	if originalSourceHash == "" {
		t.Fatalf("original SourceHash empty after add")
	}

	writeFile(t, srcPath, `def greet(name: str):
    # body changed
    print("greetings,", name)
`)
	expectedNewHash, err := utils.HashFile(srcPath)
	if err != nil {
		t.Fatalf("HashFile: %v", err)
	}
	if expectedNewHash == originalSourceHash {
		t.Fatalf("test setup error: rewritten file produced same hash")
	}

	currentSourceHash, err := utils.HashFile(originalDesc.AbsPathToSourceFile)
	if err != nil {
		t.Fatalf("HashFile current: %v", err)
	}
	if currentSourceHash == originalDesc.SourceHash {
		t.Fatalf("staleness check would not trigger; hashes still equal")
	}

	gen := extract.FunctionDescriptorGenerator{
		CommandName:         originalDesc.CommandName,
		FunctionName:        originalDesc.FunctionName,
		AbsPathToSourceFile: originalDesc.AbsPathToSourceFile,
		RelPathToSourceFile: originalDesc.RelPathToSourceFile,
		ScrollPath:          originalDesc.ScrollPath,
	}
	resolved, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	resolved.SpellHash = originalDesc.SpellHash
	if err := cache.AddFunctionDescriptor(resolved); err != nil {
		t.Fatalf("AddFunctionDescriptor: %v", err)
	}

	cache.ResetCache()
	c2, err := cache.ReadDescriptorCache(scrollPath)
	if err != nil {
		t.Fatalf("ReadDescriptorCache after refresh: %v", err)
	}
	updated := c2.Functions["greet"]
	if updated.SourceHash != expectedNewHash {
		t.Errorf("SourceHash after refresh = %q, want %q", updated.SourceHash, expectedNewHash)
	}
	if updated.SourceHash == originalSourceHash {
		t.Errorf("SourceHash unchanged after re-extraction")
	}
}

// TestStaleness_Spell mirrors the spell-hash staleness loop in
// cmd/root.go:84-118. Given a cache whose descriptor SpellHash predates an
// edit to scroll.yaml, the loop must re-extract and write back a descriptor
// whose SpellHash equals the edited spell's Hash().
func TestStaleness_Spell(t *testing.T) {
	setupTestEnv(t)
	dir := withScrollDir(t)

	srcPath := filepath.Join(dir, "greet.py")
	writeFile(t, srcPath, `def greet(name: str):
    print(name)
`)
	scrollPath := filepath.Join(dir, "scroll.yaml")
	writeFile(t, scrollPath, "spells:\n  - command: greet\n    path: greet.py\n    function: greet\n")

	scrollObj, err := scroll.ParseScroll(scrollPath)
	if err != nil {
		t.Fatalf("ParseScroll: %v", err)
	}
	originalSpellHash, err := scrollObj.Spells[0].Hash()
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}

	writeCacheDirect(t, scrollPath, map[string]desc.FunctionDescriptor{
		"greet": {
			CommandName:         "greet",
			FunctionName:        "greet",
			AbsPathToSourceFile: srcPath,
			RelPathToSourceFile: "greet.py",
			ScrollPath:          scrollPath,
			SpellHash:           originalSpellHash,
			Params: []desc.ParamDescriptor{
				{Name: "name", ResolvedType: &desc.TypeInfo{Kind: desc.TypeKindPrimitive, Name: "str"}, Default: nil},
			},
		},
	})

	writeFile(t, scrollPath, "spells:\n  - command: greet\n    path: greet.py\n    function: greet\n    interpreter: /usr/bin/python3\n")
	scroll.ResetScrollCache()
	editedScroll, err := scroll.ParseScroll(scrollPath)
	if err != nil {
		t.Fatalf("ParseScroll edited: %v", err)
	}
	newSpellHash, err := editedScroll.Spells[0].Hash()
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	if newSpellHash == originalSpellHash {
		t.Fatalf("test setup error: edited spell has the same hash as before")
	}

	cache.ResetCache()
	descriptorCache, err := cache.ReadDescriptorCache(scrollPath)
	if err != nil {
		t.Fatalf("ReadDescriptorCache: %v", err)
	}
	for _, sp := range editedScroll.Spells {
		currHash, err := sp.Hash()
		if err != nil {
			t.Fatalf("Hash: %v", err)
		}
		fd, ok := descriptorCache.Functions[sp.Command]
		if !ok {
			t.Fatalf("descriptor missing for %s", sp.Command)
		}
		if currHash == fd.SpellHash {
			continue
		}
		gen := extract.FunctionDescriptorGenerator{
			CommandName:         sp.Command,
			FunctionName:        sp.Function,
			AbsPathToSourceFile: sp.AbsPath,
			RelPathToSourceFile: sp.Path,
			ScrollPath:          editedScroll.Path,
		}
		resolved, err := gen.Generate()
		if err != nil {
			t.Fatalf("Generate: %v", err)
		}
		resolved.SpellHash = currHash
		if err := cache.AddFunctionDescriptor(resolved); err != nil {
			t.Fatalf("AddFunctionDescriptor: %v", err)
		}
	}

	c2 := readCacheDirect(t, scrollPath)
	updated := c2.Functions["greet"]
	if updated.SpellHash != newSpellHash {
		t.Errorf("SpellHash after refresh = %q, want %q", updated.SpellHash, newSpellHash)
	}
	if updated.SpellHash == originalSpellHash {
		t.Errorf("SpellHash unchanged after staleness fix")
	}
	if len(updated.Params) == 0 {
		t.Errorf("Params empty after re-extraction")
	}
}

// setupConditionalRitual writes a scroll with a binary-branching ritual and
// the three Python spells it references, reconciles the descriptor cache,
// and returns the parent cobra command the test will SetArgs against.
//
// Scroll shape:
//
//	rituals:
//	  - command: maybe
//	    steps:
//	      - id: check
//	        spell: get_status   # returns {"ok": <bool>}
//	      - if: check.ok
//	        then:
//	          - spell: handle_ok    # prints "OK"
//	        else:
//	          - spell: handle_err   # prints "ERR"
func setupConditionalRitual(t *testing.T) *cobra.Command {
	t.Helper()
	setupTestEnv(t)
	dir := withScrollDir(t)

	scrollPath := filepath.Join(dir, "scroll.yaml")
	writeFile(t, scrollPath, `spells:
  - command: get_status
    path: status.py
    function: get_status
    interpreter: python3
  - command: handle_ok
    path: handlers.py
    function: handle_ok
    interpreter: python3
  - command: handle_err
    path: handlers.py
    function: handle_err
    interpreter: python3
rituals:
  - command: maybe
    steps:
      - id: check
        spell: get_status
      - if: check.ok
        then:
          - spell: handle_ok
        else:
          - spell: handle_err
`)
	writeFile(t, filepath.Join(dir, "status.py"), `def get_status(ok: bool):
    return {"ok": ok}
`)
	writeFile(t, filepath.Join(dir, "handlers.py"), `def handle_ok():
    return "OK"

def handle_err():
    return "ERR"
`)

	s, err := scroll.ParseScroll(scrollPath)
	if err != nil {
		t.Fatalf("ParseScroll: %v", err)
	}
	dc, err := cache.ReadDescriptorCache(scrollPath)
	if err != nil {
		t.Fatalf("ReadDescriptorCache: %v", err)
	}
	// Reconciler prints "Unearthed..." progress lines — swallow.
	captureOutput(t, func() {
		if err := resolve.ReconcileScrollAndDescriptors(s, dc); err != nil {
			t.Fatalf("Reconcile: %v", err)
		}
	})

	commands, err := GenerateCommands(dc, nil, "", nil)
	if err != nil {
		t.Fatalf("GenerateCommands: %v", err)
	}
	parent := &cobra.Command{Use: "test"}
	for _, cm := range commands {
		parent.AddCommand(cm)
	}
	return parent
}

func TestRun_RitualConditional_TakesThenBranch(t *testing.T) {
	if !pythonAvailable() {
		t.Skip("python3 not on PATH")
	}
	parent := setupConditionalRitual(t)

	stdout, _ := captureOutput(t, func() {
		parent.SetArgs([]string{"maybe", "--ok"})
		if err := parent.Execute(); err != nil {
			t.Fatalf("execute: %v", err)
		}
	})
	if !strings.Contains(stdout, "OK") {
		t.Errorf("expected 'OK' in stdout (then branch), got:\n%s", stdout)
	}
	if strings.Contains(stdout, "ERR") {
		t.Errorf("expected else branch to be skipped, got:\n%s", stdout)
	}
}

func TestRun_RitualConditional_TakesElseBranch(t *testing.T) {
	if !pythonAvailable() {
		t.Skip("python3 not on PATH")
	}
	parent := setupConditionalRitual(t)

	stdout, _ := captureOutput(t, func() {
		parent.SetArgs([]string{"maybe", "--ok=false"})
		if err := parent.Execute(); err != nil {
			t.Fatalf("execute: %v", err)
		}
	})
	if !strings.Contains(stdout, "ERR") {
		t.Errorf("expected 'ERR' in stdout (else branch), got:\n%s", stdout)
	}
	if strings.Contains(stdout, "OK") {
		t.Errorf("expected then branch to be skipped, got:\n%s", stdout)
	}
}

// TestRun_RitualLetStep_BindsExpressionResult verifies that a let-step
// stores the evaluated expression into the bindings map so subsequent
// steps (here, an if-step) can reference it.
func TestRun_RitualLetStep_BindsExpressionResult(t *testing.T) {
	if !pythonAvailable() {
		t.Skip("python3 not on PATH")
	}
	setupTestEnv(t)
	dir := withScrollDir(t)

	scrollPath := filepath.Join(dir, "scroll.yaml")
	writeFile(t, scrollPath, `spells:
  - command: classify
    path: cls.py
    function: classify
    interpreter: python3
  - command: handle_extreme
    path: cls.py
    function: handle_extreme
    interpreter: python3
  - command: handle_normal
    path: cls.py
    function: handle_normal
    interpreter: python3
rituals:
  - command: branchy
    steps:
      - id: t
        spell: classify
      - let: extreme
        value: t.hot || t.cold
      - if: extreme
        then:
          - spell: handle_extreme
        else:
          - spell: handle_normal
`)
	writeFile(t, filepath.Join(dir, "cls.py"), `def classify(celsius: int):
    return {"hot": celsius >= 30, "cold": celsius <= 0}

def handle_extreme():
    return "EXTREME"

def handle_normal():
    return "NORMAL"
`)

	s, err := scroll.ParseScroll(scrollPath)
	if err != nil {
		t.Fatalf("ParseScroll: %v", err)
	}
	dc, err := cache.ReadDescriptorCache(scrollPath)
	if err != nil {
		t.Fatalf("ReadDescriptorCache: %v", err)
	}
	captureOutput(t, func() {
		if err := resolve.ReconcileScrollAndDescriptors(s, dc); err != nil {
			t.Fatalf("Reconcile: %v", err)
		}
	})
	commands, err := GenerateCommands(dc, nil, "", nil)
	if err != nil {
		t.Fatalf("GenerateCommands: %v", err)
	}
	parent := &cobra.Command{Use: "test"}
	for _, cm := range commands {
		parent.AddCommand(cm)
	}

	cases := []struct {
		name    string
		celsius string
		want    string
	}{
		{"hot", "40", "EXTREME"},
		{"cold", "-10", "EXTREME"},
		{"mild", "20", "NORMAL"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			stdout, _ := captureOutput(t, func() {
				parent.SetArgs([]string{"branchy", "--celsius", c.celsius})
				if err := parent.Execute(); err != nil {
					t.Fatalf("execute: %v", err)
				}
			})
			if !strings.Contains(stdout, c.want) {
				t.Errorf("celsius=%s: expected %q in stdout, got:\n%s", c.celsius, c.want, stdout)
			}
		})
	}
}

// TestRun_RitualConditional_PrevResultThreadsThroughBranch verifies that a
// step following an if-step auto-binds from prev_result set by the tail of
// the chosen branch (not from before the if-step).
func TestRun_RitualConditional_PrevResultThreadsThroughBranch(t *testing.T) {
	if !pythonAvailable() {
		t.Skip("python3 not on PATH")
	}
	setupTestEnv(t)
	dir := withScrollDir(t)

	scrollPath := filepath.Join(dir, "scroll.yaml")
	writeFile(t, scrollPath, `spells:
  - command: gate
    path: flow.py
    function: gate
    interpreter: python3
  - command: emit_a
    path: flow.py
    function: emit_a
    interpreter: python3
  - command: emit_b
    path: flow.py
    function: emit_b
    interpreter: python3
  - command: echo_val
    path: flow.py
    function: echo_val
    interpreter: python3
rituals:
  - command: chain
    steps:
      - id: g
        spell: gate
      - if: g.left
        then:
          - spell: emit_a
        else:
          - spell: emit_b
      - spell: echo_val
`)
	writeFile(t, filepath.Join(dir, "flow.py"), `def gate(left: bool):
    return {"left": left}

def emit_a():
    return "from-a"

def emit_b():
    return "from-b"

def echo_val(val: str):
    return f"got:{val}"
`)

	s, err := scroll.ParseScroll(scrollPath)
	if err != nil {
		t.Fatalf("ParseScroll: %v", err)
	}
	dc, err := cache.ReadDescriptorCache(scrollPath)
	if err != nil {
		t.Fatalf("ReadDescriptorCache: %v", err)
	}
	captureOutput(t, func() {
		if err := resolve.ReconcileScrollAndDescriptors(s, dc); err != nil {
			t.Fatalf("Reconcile: %v", err)
		}
	})

	commands, err := GenerateCommands(dc, nil, "", nil)
	if err != nil {
		t.Fatalf("GenerateCommands: %v", err)
	}
	parent := &cobra.Command{Use: "test"}
	for _, cm := range commands {
		parent.AddCommand(cm)
	}

	stdout, _ := captureOutput(t, func() {
		parent.SetArgs([]string{"chain", "--left"})
		if err := parent.Execute(); err != nil {
			t.Fatalf("execute (left): %v", err)
		}
	})
	if !strings.Contains(stdout, "got:from-a") {
		t.Errorf("expected echo_val to receive emit_a output via prev_result, got:\n%s", stdout)
	}
}
