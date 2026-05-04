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
				CommandName:  "greet",
				FunctionName: "greet",
				SourceFile:   "/tmp/fake/greet.py",
				ScrollPath:   "/tmp/fake/scroll.yaml",
				Params: []desc.ParamDescriptor{
					{Name: "n", ResolvedType: &desc.TypeInfo{Kind: "int", Name: "int"}, Default: 1},
					{Name: "who", ResolvedType: &desc.TypeInfo{Kind: "string", Name: "str"}, Default: nil},
				},
			},
		},
	}

	commands, err := GenerateCommands(c)
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
			CommandName:  "greet",
			FunctionName: "greet",
			SourceFile:   srcPath,
			ScrollPath:   scrollPath,
			Interpreter:  "python3",
			SourceHash:   srcHash,
			Params: []desc.ParamDescriptor{
				{Name: "name", ResolvedType: &desc.TypeInfo{Kind: "string", Name: "str"}, Default: "world"},
			},
		},
	})

	c, err := cache.ReadDescriptorCache(scrollPath)
	if err != nil {
		t.Fatalf("ReadDescriptorCache: %v", err)
	}
	commands, err := GenerateCommands(c)
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

	currentSourceHash, err := utils.HashFile(originalDesc.SourceFile)
	if err != nil {
		t.Fatalf("HashFile current: %v", err)
	}
	if currentSourceHash == originalDesc.SourceHash {
		t.Fatalf("staleness check would not trigger; hashes still equal")
	}

	gen := extract.FunctionDescriptorGenerator{
		CommandName:  originalDesc.CommandName,
		FunctionName: originalDesc.FunctionName,
		SourceFile:   originalDesc.SourceFile,
		ScrollPath:   originalDesc.ScrollPath,
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
			CommandName:  "greet",
			FunctionName: "greet",
			SourceFile:   srcPath,
			ScrollPath:   scrollPath,
			SpellHash:    originalSpellHash,
			Params: []desc.ParamDescriptor{
				{Name: "name", ResolvedType: &desc.TypeInfo{Kind: "string", Name: "str"}, Default: nil},
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
		fd, ok := descriptorCache.Functions[sp.Function]
		if !ok {
			t.Fatalf("descriptor missing for %s", sp.Function)
		}
		if currHash == fd.SpellHash {
			continue
		}
		gen := extract.FunctionDescriptorGenerator{
			CommandName:  sp.Command,
			FunctionName: sp.Function,
			SourceFile:   sp.AbsPath,
			ScrollPath:   editedScroll.Path,
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
