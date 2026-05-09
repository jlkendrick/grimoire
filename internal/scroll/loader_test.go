package scroll_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	scroll "github.com/jlkendrick/grimoire/internal/scroll"
	ts "github.com/jlkendrick/grimoire/internal/testsupport"
)

// writeRegistry overwrites ~/.grimoire/grimoire.yaml under the test's
// GRIMOIRE_HOME to register the given absolute scroll paths.
func writeRegistry(t *testing.T, home string, scrollPaths []string) {
	t.Helper()
	body := "registered_scrolls:\n"
	for _, p := range scrollPaths {
		body += fmt.Sprintf("  - path: %s\n", p)
	}
	ts.WriteFile(t, filepath.Join(home, "grimoire.yaml"), body)
}

func TestLoadScrolls_LocalWins(t *testing.T) {
	home := ts.SetupGrimoireHome(t)
	dir := ts.WithScrollDir(t)

	// Local scroll in cwd.
	localPath := ts.WriteScrollYAML(t, dir, "spells:\n  - command: local_only\n    path: x.py\n    function: x\n")

	// Also register a *different* scroll in the global registry.
	otherDir := t.TempDir()
	otherDirAbs, err := filepath.EvalSymlinks(otherDir)
	if err != nil {
		t.Fatalf("EvalSymlinks: %v", err)
	}
	otherPath := ts.WriteScrollYAML(t, otherDirAbs, "spells:\n  - command: global_only\n    path: y.py\n    function: y\n")
	writeRegistry(t, home, []string{otherPath})

	scrolls, err := scroll.LoadScrolls()
	if err != nil {
		t.Fatalf("LoadScrolls: %v", err)
	}
	if len(scrolls) != 1 {
		t.Fatalf("expected exactly 1 scroll (the local one), got %d", len(scrolls))
	}
	if scrolls[0].Path != localPath {
		t.Errorf("Path = %q, want %q (local should win)", scrolls[0].Path, localPath)
	}
	if len(scrolls[0].Spells) != 1 || scrolls[0].Spells[0].Command != "local_only" {
		t.Errorf("expected local scroll content, got Spells=%+v", scrolls[0].Spells)
	}
}

func TestLoadScrolls_FallsBackToGlobal(t *testing.T) {
	home := ts.SetupGrimoireHome(t)
	// chdir somewhere with no scroll.yaml above it.
	cwd := ts.WithScrollDir(t)
	if _, err := os.Stat(filepath.Join(cwd, "scroll.yaml")); err == nil {
		t.Fatalf("test setup error: scroll.yaml unexpectedly present in cwd")
	}

	a := t.TempDir()
	aAbs, err := filepath.EvalSymlinks(a)
	if err != nil {
		t.Fatalf("EvalSymlinks: %v", err)
	}
	b := t.TempDir()
	bAbs, err := filepath.EvalSymlinks(b)
	if err != nil {
		t.Fatalf("EvalSymlinks: %v", err)
	}
	pathA := ts.WriteScrollYAML(t, aAbs, "spells:\n  - command: a_cmd\n    path: a.py\n    function: a\n")
	pathB := ts.WriteScrollYAML(t, bAbs, "spells:\n  - command: b_cmd\n    path: b.py\n    function: b\n")
	writeRegistry(t, home, []string{pathA, pathB})

	scrolls, err := scroll.LoadScrolls()
	if err != nil {
		t.Fatalf("LoadScrolls: %v", err)
	}
	if len(scrolls) != 2 {
		t.Fatalf("expected 2 scrolls from registry, got %d", len(scrolls))
	}
	if scrolls[0].Path != pathA || scrolls[1].Path != pathB {
		t.Errorf("registry order not preserved: got [%s, %s]", scrolls[0].Path, scrolls[1].Path)
	}
}

func TestLoadScrolls_Memoizes(t *testing.T) {
	ts.SetupGrimoireHome(t)
	dir := ts.WithScrollDir(t)
	ts.WriteScrollYAML(t, dir, "spells:\n  - command: c\n    path: c.py\n    function: c\n")

	first, err := scroll.LoadScrolls()
	if err != nil {
		t.Fatalf("LoadScrolls 1: %v", err)
	}
	second, err := scroll.LoadScrolls()
	if err != nil {
		t.Fatalf("LoadScrolls 2: %v", err)
	}
	if len(first) != 1 || len(second) != 1 {
		t.Fatalf("unexpected lengths: %d / %d", len(first), len(second))
	}
	if first[0] != second[0] {
		t.Errorf("expected memoized identity, got distinct pointers")
	}
}

func TestResetScrollCache_Clears(t *testing.T) {
	ts.SetupGrimoireHome(t)
	dir := ts.WithScrollDir(t)
	path := ts.WriteScrollYAML(t, dir, "spells:\n  - command: c\n    path: c.py\n    function: c\n")

	first, err := scroll.LoadScrolls()
	if err != nil {
		t.Fatalf("LoadScrolls 1: %v", err)
	}

	// Mutate the file, then reset the memo and reload.
	ts.WriteFile(t, path, "spells:\n  - command: c2\n    path: c.py\n    function: c\n")
	scroll.ResetScrollCache()

	second, err := scroll.LoadScrolls()
	if err != nil {
		t.Fatalf("LoadScrolls 2: %v", err)
	}
	if first[0] == second[0] {
		t.Errorf("expected fresh pointer after reset, got same")
	}
	if second[0].Spells[0].Command != "c2" {
		t.Errorf("expected reload to see edited Command 'c2', got %q", second[0].Spells[0].Command)
	}
}

func TestFindLocalScroll_UpwardsTraversal(t *testing.T) {
	root := t.TempDir()
	rootAbs, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatalf("EvalSymlinks: %v", err)
	}
	scrollPath := filepath.Join(rootAbs, "scroll.yaml")
	ts.WriteFile(t, scrollPath, "spells: []\n")

	deep := filepath.Join(rootAbs, "a", "b", "c")
	if err := os.MkdirAll(deep, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	got, found := scroll.FindLocalScroll(deep)
	if !found {
		t.Fatalf("FindLocalScroll returned not-found from %s", deep)
	}
	if got != scrollPath {
		t.Errorf("path = %q, want %q", got, scrollPath)
	}
}
