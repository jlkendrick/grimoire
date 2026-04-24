package cmd

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	core "github.com/jlkendrick/grimoire/core"
)

// withTempGrimoireHome creates a temp dir with a minimal grimoire.yaml, sets
// GRIMOIRE_HOME to that dir, and returns the config path. The env var and
// config cache are both restored automatically when the test ends.
func withTempGrimoireHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	configPath := filepath.Join(dir, "grimoire.yaml")
	if err := os.WriteFile(configPath, []byte("{}\n"), 0644); err != nil {
		t.Fatalf("creating temp grimoire.yaml: %v", err)
	}
	t.Setenv("GRIMOIRE_HOME", dir)
	t.Cleanup(core.ResetConfigCache)
	return configPath
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

		if !strings.Contains(output, "no scroll.yaml file found") {
			t.Errorf("expected 'no scroll.yaml file found' in output, got: %q", output)
		}
	})

	t.Run("explicit_path_registers_project", func(t *testing.T) {
		configPath := withTempGrimoireHome(t)

		dir := t.TempDir()
		scrollPath := filepath.Join(dir, "scroll.yaml")
		if err := os.WriteFile(scrollPath, []byte("{}\n"), 0644); err != nil {
			t.Fatal(err)
		}

		output := captureStdout(t, func() {
			rootCmd.SetArgs([]string{"register", scrollPath})
			_ = rootCmd.Execute()
		})

		if !strings.Contains(output, "registered with the global grimoire") {
			t.Errorf("expected success message in output, got: %q", output)
		}

		updated, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("reading updated global config: %v", err)
		}
		if !strings.Contains(string(updated), scrollPath) {
			t.Errorf("expected %q in global config, got:\n%s", scrollPath, string(updated))
		}
	})

	t.Run("traversal_finds_spell_yaml", func(t *testing.T) {
		withTempGrimoireHome(t)

		// Create a parent with scroll.yaml and a child subdir to run from.
		parent := t.TempDir()
		scrollPath := filepath.Join(parent, "scroll.yaml")
		if err := os.WriteFile(scrollPath, []byte("{}\n"), 0644); err != nil {
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
		if !strings.Contains(output, scrollPath) {
			t.Errorf("expected spell path %q in output, got: %q", scrollPath, output)
		}
	})
}
