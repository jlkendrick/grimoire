package cmd

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	core "github.com/jlkendrick/grimoire/core"
)

// globalConfigPath returns the hardcoded path to the global grimoire config,
// skipping the test if the home directory cannot be resolved.
func globalConfigPath(t *testing.T) string {
	t.Helper()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("could not get home dir: %v", err)
	}
	return filepath.Join(home, "Code/Projects/grimoire/.grimoire/grimoire.yaml")
}

// withGlobalConfig reads the current global config file and registers a cleanup
// that restores it at the end of the test. Skips if the file is unavailable.
func withGlobalConfig(t *testing.T) {
	t.Helper()
	path := globalConfigPath(t)
	original, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("global config not available at %s: %v", path, err)
	}
	t.Cleanup(func() {
		if err := os.WriteFile(path, original, 0644); err != nil {
			t.Logf("warning: could not restore global config: %v", err)
		}
	})
}

func TestRegisterCmd(t *testing.T) {
	rootCmd.SetErr(io.Discard)

	t.Run("no_spell_yaml_prints_error", func(t *testing.T) {
		t.Cleanup(core.ResetConfigCache)
		dir := t.TempDir()
		origDir, _ := os.Getwd()
		defer os.Chdir(origDir)
		if err := os.Chdir(dir); err != nil {
			t.Fatal(err)
		}

		output := captureStdout(t, func() {
			rootCmd.SetArgs([]string{"register"})
			_ = rootCmd.Execute()
		})

		if !strings.Contains(output, "no spell.yaml file found") {
			t.Errorf("expected 'no spell.yaml file found' in output, got: %q", output)
		}
	})

	t.Run("explicit_path_registers_project", func(t *testing.T) {
		t.Cleanup(core.ResetConfigCache)
		withGlobalConfig(t)

		dir := t.TempDir()
		spellPath := filepath.Join(dir, "spell.yaml")
		if err := os.WriteFile(spellPath, []byte("{}\n"), 0644); err != nil {
			t.Fatal(err)
		}

		output := captureStdout(t, func() {
			rootCmd.SetArgs([]string{"register", spellPath})
			_ = rootCmd.Execute()
		})

		if !strings.Contains(output, "registered with the global grimoire") {
			t.Errorf("expected success message in output, got: %q", output)
		}

		updated, err := os.ReadFile(globalConfigPath(t))
		if err != nil {
			t.Fatalf("reading updated global config: %v", err)
		}
		if !strings.Contains(string(updated), spellPath) {
			t.Errorf("expected %q in global config, got:\n%s", spellPath, string(updated))
		}
	})

	t.Run("traversal_finds_spell_yaml", func(t *testing.T) {
		t.Cleanup(core.ResetConfigCache)
		withGlobalConfig(t)

		// Create a parent with spell.yaml and a child subdir to run from.
		parent := t.TempDir()
		spellPath := filepath.Join(parent, "spell.yaml")
		if err := os.WriteFile(spellPath, []byte("{}\n"), 0644); err != nil {
			t.Fatal(err)
		}
		child := filepath.Join(parent, "subdir")
		if err := os.Mkdir(child, 0755); err != nil {
			t.Fatal(err)
		}

		origDir, _ := os.Getwd()
		defer os.Chdir(origDir)
		if err := os.Chdir(child); err != nil {
			t.Fatal(err)
		}

		output := captureStdout(t, func() {
			rootCmd.SetArgs([]string{"register"})
			_ = rootCmd.Execute()
		})

		if !strings.Contains(output, "registered with the global grimoire") {
			t.Errorf("expected success message in output, got: %q", output)
		}
		if !strings.Contains(output, spellPath) {
			t.Errorf("expected spell path %q in output, got: %q", spellPath, output)
		}
	})
}
