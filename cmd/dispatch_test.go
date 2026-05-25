package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	scroll "github.com/jlkendrick/grimoire/internal/scroll"
)

// withFreshRoot swaps in a clean rootCmd for the duration of a dispatch test
// so registrations don't leak between tests, and restores the original on
// cleanup.
func withFreshRoot(t *testing.T) {
	t.Helper()
	original := rootCmd
	rootCmd = &cobra.Command{Use: "grimoire"}
	t.Cleanup(func() { rootCmd = original })
}

// pythonSpell writes a minimal Python file and returns a YAML fragment for a
// spell entry referencing it. The function body is irrelevant since dispatch
// tests never actually run the spell.
func pythonSpell(t *testing.T, dir, command, function string) string {
	t.Helper()
	src := function + ".py"
	writeFile(t, filepath.Join(dir, src), "def "+function+"():\n    return None\n")
	return "  - command: " + command + "\n    path: " + src + "\n    function: " + function + "\n"
}

func registeredUses() map[string]bool {
	uses := map[string]bool{}
	for _, c := range rootCmd.Commands() {
		uses[c.Use] = true
	}
	return uses
}

// TestRegisterScrollCommands_NoCollisions confirms that two scrolls with
// distinct commands both register with bare names, no dot-form.
func TestRegisterScrollCommands_NoCollisions(t *testing.T) {
	setupTestEnv(t)
	withFreshRoot(t)

	dirA := filepath.Join(t.TempDir(), "projA")
	dirB := filepath.Join(t.TempDir(), "projB")
	for _, d := range []string{dirA, dirB} {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatalf("MkdirAll %s: %v", d, err)
		}
	}
	pathA := filepath.Join(dirA, "scroll.yaml")
	pathB := filepath.Join(dirB, "scroll.yaml")
	writeFile(t, pathA, "name: projA\nspells:\n"+pythonSpell(t, dirA, "alpha", "alpha"))
	writeFile(t, pathB, "name: projB\nspells:\n"+pythonSpell(t, dirB, "beta", "beta"))

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

	uses := registeredUses()
	if !uses["alpha"] {
		t.Errorf("bare 'alpha' should be registered; got %v", uses)
	}
	if !uses["beta"] {
		t.Errorf("bare 'beta' should be registered; got %v", uses)
	}
	if uses["projA.alpha"] || uses["projB.beta"] {
		t.Errorf("dot-form should NOT be registered when no collision; got %v", uses)
	}
}

// TestRegisterScrollCommands_DuplicateCommand confirms that when two scrolls
// share a command name, both get dot-form registrations and the bare form is
// not registered.
func TestRegisterScrollCommands_DuplicateCommand(t *testing.T) {
	setupTestEnv(t)
	withFreshRoot(t)

	dirA := filepath.Join(t.TempDir(), "projA")
	dirB := filepath.Join(t.TempDir(), "projB")
	for _, d := range []string{dirA, dirB} {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
	}
	pathA := filepath.Join(dirA, "scroll.yaml")
	pathB := filepath.Join(dirB, "scroll.yaml")
	writeFile(t, pathA, "name: projA\nspells:\n"+pythonSpell(t, dirA, "deploy", "deploy"))
	writeFile(t, pathB, "name: projB\nspells:\n"+pythonSpell(t, dirB, "deploy", "deploy"))

	sA, _ := scroll.ParseScroll(pathA)
	sB, _ := scroll.ParseScroll(pathB)

	if err := registerScrollCommands([]*scroll.Scroll{sA, sB}); err != nil {
		t.Fatalf("registerScrollCommands: %v", err)
	}

	uses := registeredUses()
	if uses["deploy"] {
		t.Errorf("bare 'deploy' should NOT be registered when colliding; got %v", uses)
	}
	if !uses["projA.deploy"] {
		t.Errorf("'projA.deploy' should be registered; got %v", uses)
	}
	if !uses["projB.deploy"] {
		t.Errorf("'projB.deploy' should be registered; got %v", uses)
	}
}

// TestRegisterScrollCommands_ReservedWordForcesDotForm confirms that a spell
// whose command matches a reserved static command (`init`) only gets the
// dot-form registration, never bare.
func TestRegisterScrollCommands_ReservedWordForcesDotForm(t *testing.T) {
	setupTestEnv(t)
	withFreshRoot(t)

	dir := filepath.Join(t.TempDir(), "tools")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	scrollPath := filepath.Join(dir, "scroll.yaml")
	writeFile(t, scrollPath, "name: tools\nspells:\n"+pythonSpell(t, dir, "init", "init_fn"))

	s, err := scroll.ParseScroll(scrollPath)
	if err != nil {
		t.Fatalf("ParseScroll: %v", err)
	}

	if err := registerScrollCommands([]*scroll.Scroll{s}); err != nil {
		t.Fatalf("registerScrollCommands: %v", err)
	}

	uses := registeredUses()
	if uses["init"] {
		t.Errorf("bare 'init' should not be registered (reserved); got %v", uses)
	}
	if !uses["tools.init"] {
		t.Errorf("'tools.init' should be registered; got %v", uses)
	}
}

// TestRegisterScrollCommands_SameNameShadowsWithWarning confirms that two
// scrolls sharing the same Name and a colliding command produce a stderr
// warning and first-wins on the resolved dot-form.
func TestRegisterScrollCommands_SameNameShadowsWithWarning(t *testing.T) {
	setupTestEnv(t)
	withFreshRoot(t)

	dirA := filepath.Join(t.TempDir(), "siteA")
	dirB := filepath.Join(t.TempDir(), "siteB")
	for _, d := range []string{dirA, dirB} {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
	}
	pathA := filepath.Join(dirA, "scroll.yaml")
	pathB := filepath.Join(dirB, "scroll.yaml")
	writeFile(t, pathA, "name: proj\nspells:\n"+pythonSpell(t, dirA, "deploy", "deploy"))
	writeFile(t, pathB, "name: proj\nspells:\n"+pythonSpell(t, dirB, "deploy", "deploy"))

	sA, _ := scroll.ParseScroll(pathA)
	sB, _ := scroll.ParseScroll(pathB)

	var stderr string
	_, stderr = captureOutput(t, func() {
		if err := registerScrollCommands([]*scroll.Scroll{sA, sB}); err != nil {
			t.Fatalf("registerScrollCommands: %v", err)
		}
	})

	if !strings.Contains(stderr, "shadowed") {
		t.Errorf("expected stderr to contain shadow warning; got %q", stderr)
	}
	uses := registeredUses()
	if !uses["proj.deploy"] {
		t.Errorf("'proj.deploy' should be registered (first-wins); got %v", uses)
	}
	if uses["deploy"] {
		t.Errorf("bare 'deploy' should not be registered; got %v", uses)
	}
}

// TestStaticCommandShortCircuit_ExactMatchOnly pins down that os.Args[1] of
// the form `init.foo` does NOT short-circuit the static-command path,
// confirming dot-form invocations of reserved-name spells reach dynamic
// dispatch.
func TestStaticCommandShortCircuit_ExactMatchOnly(t *testing.T) {
	if _, ok := staticCommands["init"]; !ok {
		t.Fatalf("test premise broken: 'init' is no longer in staticCommands")
	}
	if _, ok := staticCommands["init.foo"]; ok {
		t.Errorf("'init.foo' must NOT be in staticCommands; otherwise dot-form invocations short-circuit")
	}
}

