package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	scroll "github.com/jlkendrick/grimoire/internal/scroll"
)

// makeProjectDir creates a directory under t.TempDir() and returns its
// absolute path. Helps keep multi-scroll fixtures readable.
func makeProjectDir(t *testing.T, name string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("MkdirAll %s: %v", dir, err)
	}
	return dir
}

// TestCrossScrollRitualRef_HappyPath sets up scroll A (name=a) with a
// ritual that references `b.deploy`, and scroll B (name=b) with a spell
// `deploy`. After registerScrollCommands, scroll A's cache should contain
// the resolved PipelineDescriptor with the dotted SpellName preserved.
func TestCrossScrollRitualRef_HappyPath(t *testing.T) {
	home := setupTestEnv(t)
	withFreshRoot(t)

	dirA := makeProjectDir(t, "projA")
	dirB := makeProjectDir(t, "projB")
	pathA := filepath.Join(dirA, "scroll.yaml")
	pathB := filepath.Join(dirB, "scroll.yaml")

	writeFile(t, pathB, "name: b\nspells:\n"+pythonSpell(t, dirB, "deploy", "deploy"))
	writeFile(t, pathA,
		"name: a\nspells:\n"+pythonSpell(t, dirA, "seed", "seed")+
			"rituals:\n  - command: pipeline\n    steps:\n      - spell: seed\n      - spell: b.deploy\n")

	// Register both scrolls globally.
	registry := &scroll.Scroll{
		RegisteredScrolls: []scroll.ScrollPath{{Path: pathA}, {Path: pathB}},
		Path:              filepath.Join(home, "grimoire.yaml"),
	}
	if err := registry.Write(); err != nil {
		t.Fatalf("write registry: %v", err)
	}

	sA, err := scroll.ParseScroll(pathA)
	if err != nil {
		t.Fatalf("ParseScroll A: %v", err)
	}
	sB, err := scroll.ParseScroll(pathB)
	if err != nil {
		t.Fatalf("ParseScroll B: %v", err)
	}

	if err := registerScrollCommands([]*scroll.Scroll{sA, sB}); err != nil {
		t.Fatalf("registerScrollCommands: %v", err)
	}

	// scroll A's pipeline should be registered (no command collision with B)
	// and the underlying PipelineDescriptor should contain the dotted step.
	uses := registeredUses()
	if !uses["pipeline"] {
		t.Errorf("'pipeline' should be registered; got %v", uses)
	}
}

// TestCrossScrollRitualRef_UnknownModule confirms that a ritual referencing
// a non-existent scroll name errors out at reconcile.
func TestCrossScrollRitualRef_UnknownModule(t *testing.T) {
	home := setupTestEnv(t)
	withFreshRoot(t)

	dirA := makeProjectDir(t, "projA")
	pathA := filepath.Join(dirA, "scroll.yaml")
	writeFile(t, pathA,
		"name: a\nspells:\n"+pythonSpell(t, dirA, "seed", "seed")+
			"rituals:\n  - command: bad\n    steps:\n      - spell: seed\n      - spell: zzz.deploy\n")

	registry := &scroll.Scroll{
		RegisteredScrolls: []scroll.ScrollPath{{Path: pathA}},
		Path:              filepath.Join(home, "grimoire.yaml"),
	}
	if err := registry.Write(); err != nil {
		t.Fatalf("write registry: %v", err)
	}
	sA, err := scroll.ParseScroll(pathA)
	if err != nil {
		t.Fatalf("ParseScroll A: %v", err)
	}

	err = registerScrollCommands([]*scroll.Scroll{sA})
	if err == nil {
		t.Fatalf("expected reconcile error for unknown module")
	}
	if !strings.Contains(err.Error(), "no scroll named") {
		t.Errorf("error = %v, want unknown-module message", err)
	}
}

// TestCrossScrollRitualRef_LocalModeStillResolvesCrossScroll confirms that
// even when running in local mode (one active scroll discovered in cwd),
// cross-scroll refs to other registered scrolls still resolve. This
// exercises the LoadActiveAndAll / loadAllForActive union behavior.
func TestCrossScrollRitualRef_LocalModeStillResolvesCrossScroll(t *testing.T) {
	home := setupTestEnv(t)
	withFreshRoot(t)

	dirA := withScrollDir(t) // cwd is dirA
	dirB := makeProjectDir(t, "projB")
	pathA := filepath.Join(dirA, "scroll.yaml")
	pathB := filepath.Join(dirB, "scroll.yaml")

	writeFile(t, pathB, "name: b\nspells:\n"+pythonSpell(t, dirB, "deploy", "deploy"))
	writeFile(t, pathA,
		"name: a\nspells:\n"+pythonSpell(t, dirA, "seed", "seed")+
			"rituals:\n  - command: pipeline\n    steps:\n      - spell: seed\n      - spell: b.deploy\n")

	// Register scroll B globally; scroll A is discovered as local.
	registry := &scroll.Scroll{
		RegisteredScrolls: []scroll.ScrollPath{{Path: pathB}},
		Path:              filepath.Join(home, "grimoire.yaml"),
	}
	if err := registry.Write(); err != nil {
		t.Fatalf("write registry: %v", err)
	}

	// LoadActiveAndAll should detect A as local + B in registry.
	active, all, err := scroll.LoadActiveAndAll()
	if err != nil {
		t.Fatalf("LoadActiveAndAll: %v", err)
	}
	if len(active) != 1 || active[0].Path != pathA {
		t.Errorf("active = %v, want [%s]", active, pathA)
	}
	if len(all) != 2 {
		t.Errorf("all len = %d, want 2 (A local + B registered); got %v", len(all), all)
	}

	// Register only A (the local active scroll). Reconcile must still
	// resolve `b.deploy` even though B isn't in `active`.
	if err := registerScrollCommands(active); err != nil {
		t.Fatalf("registerScrollCommands: %v", err)
	}

	uses := registeredUses()
	if !uses["pipeline"] {
		t.Errorf("'pipeline' should be registered (local mode); got %v", uses)
	}
	// B's commands must NOT be registered — B is in `all` for index purposes
	// only, not in `active`.
	if uses["deploy"] || uses["b.deploy"] {
		t.Errorf("B's commands must not be registered in local mode; got %v", uses)
	}
}
