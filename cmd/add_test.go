package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	cache "github.com/jlkendrick/grimoire/internal/cache"
	scroll "github.com/jlkendrick/grimoire/internal/scroll"
	utils "github.com/jlkendrick/grimoire/internal/utils"
)

const greetSource = `def greet(n: int = 1, who: str):
    pass
`

// intDefaultEquals reports whether v represents want. Defaults are stored
// as typed Go values; extraction produces int64, and the JSON cache
// round-trip (encoding/json decoding into any-typed fields) collapses
// numbers into float64.
func intDefaultEquals(v any, want int) bool {
	switch x := v.(type) {
	case int:
		return x == want
	case int64:
		return int(x) == want
	case float64:
		return int(x) == want && float64(int(x)) == x
	default:
		return false
	}
}

func TestAdd_BasicFlow(t *testing.T) {
	home := setupTestEnv(t)
	dir := withScrollDir(t)
	resetRootCmdState(t)

	srcPath := filepath.Join(dir, "greet.py")
	writeFile(t, srcPath, greetSource)

	stdout, _ := captureOutput(t, func() {
		rootCmd.SetArgs([]string{"add", "greet.py:greet"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("rootCmd.Execute: %v", err)
		}
	})

	if !strings.Contains(stdout, "function") {
		t.Errorf("expected stdout to mention 'function' summary line, got:\n%s", stdout)
	}

	// scroll.yaml exists and parses
	scrollPath := filepath.Join(dir, "scroll.yaml")
	if _, err := os.Stat(scrollPath); err != nil {
		t.Fatalf("scroll.yaml not created: %v", err)
	}
	scroll.ResetScrollCache()
	parsed, err := scroll.ParseScroll(scrollPath)
	if err != nil {
		t.Fatalf("ParseScroll: %v", err)
	}
	if len(parsed.Spells) != 1 {
		t.Fatalf("expected 1 spell, got %d: %+v", len(parsed.Spells), parsed.Spells)
	}
	sp := parsed.Spells[0]
	if sp.Command != "greet" {
		t.Errorf("Spell.Command = %q, want %q", sp.Command, "greet")
	}
	if sp.Function != "greet" {
		t.Errorf("Spell.Function = %q, want %q", sp.Function, "greet")
	}
	if !strings.HasSuffix(sp.Path, "greet.py") {
		t.Errorf("Spell.Path = %q, want suffix %q", sp.Path, "greet.py")
	}

	// Cache file exists under $GRIMOIRE_HOME/cache and decodes correctly
	cacheDir := filepath.Join(home, "cache")
	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		t.Fatalf("ReadDir cache: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 cache file, got %d", len(entries))
	}

	cache.ResetCache()
	descCache, err := cache.ReadDescriptorCache(scrollPath)
	if err != nil {
		t.Fatalf("ReadDescriptorCache: %v", err)
	}
	desc, ok := descCache.Functions["greet"]
	if !ok {
		t.Fatalf("descriptor for 'greet' not in cache; cache has: %v", descCache.Functions)
	}

	// Two params: n (int with default "1") and who (str, no default)
	if len(desc.Params) != 2 {
		t.Fatalf("expected 2 params, got %d: %+v", len(desc.Params), desc.Params)
	}
	byName := map[string]int{}
	for i, p := range desc.Params {
		byName[p.Name] = i
	}
	nIdx, ok := byName["n"]
	if !ok {
		t.Fatalf("param 'n' missing")
	}
	pn := desc.Params[nIdx]
	if pn.ResolvedType == nil || pn.ResolvedType.Name != "int" {
		t.Errorf("param n: ResolvedType = %+v, want Name=int", pn.ResolvedType)
	}
	if !intDefaultEquals(pn.Default, 1) {
		t.Errorf("param n: Default = %v (%T), want 1 (int, int64, or float64 after JSON round-trip)", pn.Default, pn.Default)
	}
	whoIdx, ok := byName["who"]
	if !ok {
		t.Fatalf("param 'who' missing")
	}
	pw := desc.Params[whoIdx]
	if pw.ResolvedType == nil || pw.ResolvedType.Name != "str" {
		t.Errorf("param who: ResolvedType = %+v, want Name=str", pw.ResolvedType)
	}
	if pw.Default != nil {
		t.Errorf("param who: Default = %v, want nil", pw.Default)
	}

	// Per-descriptor SourceHash matches the source file content hash.
	wantSrcHash, err := utils.HashFile(srcPath)
	if err != nil {
		t.Fatalf("HashFile: %v", err)
	}
	if desc.SourceHash != wantSrcHash {
		t.Errorf("descriptor.SourceHash = %q, want %q", desc.SourceHash, wantSrcHash)
	}

	// SpellHash matches the on-disk spell's Hash() -- this is the baseline
	// the runtime staleness check compares against.
	wantSpellHash, err := sp.Hash()
	if err != nil {
		t.Fatalf("sp.Hash: %v", err)
	}
	if desc.SpellHash != wantSpellHash {
		t.Errorf("descriptor.SpellHash = %q, want %q (matches spell.Hash())", desc.SpellHash, wantSpellHash)
	}
}

func TestAdd_RejectsInvalidFormat(t *testing.T) {
	home := setupTestEnv(t)
	dir := withScrollDir(t)
	resetRootCmdState(t)

	stdout, _ := captureOutput(t, func() {
		rootCmd.SetArgs([]string{"add", "greet.py"}) // no colon
		_ = rootCmd.Execute()
	})

	if !strings.Contains(stdout, "path_to_function:function_name format is required") {
		t.Errorf("expected format-required error, got stdout:\n%s", stdout)
	}

	if _, err := os.Stat(filepath.Join(dir, "scroll.yaml")); !os.IsNotExist(err) {
		t.Errorf("scroll.yaml should not exist after invalid input, stat err=%v", err)
	}
	entries, err := os.ReadDir(filepath.Join(home, "cache"))
	if err != nil {
		t.Fatalf("ReadDir cache: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected no cache files written, got %d", len(entries))
	}
}
