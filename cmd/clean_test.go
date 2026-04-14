package cmd

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	core "github.com/jlkendrick/grimoire/core"
)

// envsRoot returns the hardcoded grimoire envs directory and skips the test if
// it does not exist on the current machine.
func envsRoot(t *testing.T) string {
	t.Helper()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("could not get home dir: %v", err)
	}
	dir := filepath.Join(home, "Code/Projects/grimoire/.grimoire/envs")
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("envs directory not available at %s", dir)
	}
	return dir
}

// requireAllVenvsHaveOrigin skips the test if any existing venv directory in
// root (other than those named in skip) is missing its .grimoire_origin file.
// clean.go errors out immediately on a missing origin file, which would make
// deletion counts unreliable.
func requireAllVenvsHaveOrigin(t *testing.T, root string, skip ...string) {
	t.Helper()
	skipSet := map[string]bool{}
	for _, s := range skip {
		skipSet[s] = true
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("reading envs dir: %v", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() || skipSet[entry.Name()] {
			continue
		}
		origin := filepath.Join(root, entry.Name(), ".grimoire_origin")
		if _, err := os.Stat(origin); os.IsNotExist(err) {
			t.Skipf("existing venv %q has no .grimoire_origin — skipping to avoid false failure", entry.Name())
		}
	}
}

func TestCleanCmd(t *testing.T) {
	rootCmd.SetErr(io.Discard)

	// deletes_venv_for_unused_function: when a configured function's source file
	// no longer exists, the venv whose .grimoire_origin points to that file gets
	// removed.
	t.Run("deletes_venv_for_unused_function", func(t *testing.T) {
		t.Cleanup(core.ResetConfigCache)
		root := envsRoot(t)

		// Build a local spell.yaml whose function source does NOT exist.
		dir := t.TempDir()
		// Resolve symlinks so the path matches what os.Getwd() returns inside
		// the command (on macOS /var/folders/... → /private/var/folders/...).
		resolvedDir, err := filepath.EvalSymlinks(dir)
		if err != nil {
			t.Fatalf("EvalSymlinks: %v", err)
		}
		unusedPath := filepath.Join(resolvedDir, "scripts", "gone.py")

		grimContent := "functions:\n- name: gone\n  path: scripts/gone.py\n  function: gone\n"
		if err := os.WriteFile(filepath.Join(dir, "spell.yaml"), []byte(grimContent), 0644); err != nil {
			t.Fatal(err)
		}

		// Create a fake venv whose origin matches the unused function path.
		fakeVenv, err := os.MkdirTemp(root, "test_clean_del_")
		if err != nil {
			t.Fatalf("creating fake venv: %v", err)
		}
		t.Cleanup(func() { os.RemoveAll(fakeVenv) }) // no-op if already deleted

		if err := os.WriteFile(filepath.Join(fakeVenv, ".grimoire_origin"), []byte(unusedPath), 0644); err != nil {
			t.Fatalf("writing origin file: %v", err)
		}

		requireAllVenvsHaveOrigin(t, root, filepath.Base(fakeVenv))

		origDir, _ := os.Getwd()
		defer os.Chdir(origDir)
		if err := os.Chdir(dir); err != nil {
			t.Fatal(err)
		}

		output := captureStdout(t, func() {
			rootCmd.SetArgs([]string{"clean"})
			_ = rootCmd.Execute()
		})

		if !strings.Contains(output, "Deleted 1 unused venvs") {
			t.Errorf("expected 'Deleted 1 unused venvs' in output, got: %q", output)
		}
		if _, err := os.Stat(fakeVenv); !os.IsNotExist(err) {
			t.Errorf("expected fake venv %s to be deleted, but it still exists", fakeVenv)
		}
	})

	// preserves_venv_for_active_function: a venv whose origin points to a
	// function source that still exists must not be deleted.
	t.Run("preserves_venv_for_active_function", func(t *testing.T) {
		t.Cleanup(core.ResetConfigCache)
		root := envsRoot(t)

		dir := t.TempDir()
		resolvedDir, err := filepath.EvalSymlinks(dir)
		if err != nil {
			t.Fatalf("EvalSymlinks: %v", err)
		}
		existingScript := filepath.Join(resolvedDir, "live.py")
		if err := os.WriteFile(existingScript, []byte("def live(): pass\n"), 0644); err != nil {
			t.Fatal(err)
		}

		grimContent := "functions:\n- name: live\n  path: live.py\n  function: live\n"
		if err := os.WriteFile(filepath.Join(resolvedDir, "spell.yaml"), []byte(grimContent), 0644); err != nil {
			t.Fatal(err)
		}

		fakeVenv, err := os.MkdirTemp(root, "test_clean_keep_")
		if err != nil {
			t.Fatalf("creating fake venv: %v", err)
		}
		t.Cleanup(func() { os.RemoveAll(fakeVenv) })

		if err := os.WriteFile(filepath.Join(fakeVenv, ".grimoire_origin"), []byte(existingScript), 0644); err != nil {
			t.Fatalf("writing origin file: %v", err)
		}

		requireAllVenvsHaveOrigin(t, root, filepath.Base(fakeVenv))

		origDir, _ := os.Getwd()
		defer os.Chdir(origDir)
		if err := os.Chdir(resolvedDir); err != nil {
			t.Fatal(err)
		}

		output := captureStdout(t, func() {
			rootCmd.SetArgs([]string{"clean"})
			_ = rootCmd.Execute()
		})

		if !strings.Contains(output, "Deleted 0 unused venvs") {
			t.Errorf("expected 'Deleted 0 unused venvs' in output, got: %q", output)
		}
		if _, err := os.Stat(fakeVenv); err != nil {
			t.Errorf("expected fake venv %s to still exist, but: %v", fakeVenv, err)
		}
	})

	// no_functions_deletes_zero: an empty spell.yaml produces zero deletions.
	t.Run("no_functions_deletes_zero", func(t *testing.T) {
		t.Cleanup(core.ResetConfigCache)
		root := envsRoot(t)

		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "spell.yaml"), []byte("{}\n"), 0644); err != nil {
			t.Fatal(err)
		}

		requireAllVenvsHaveOrigin(t, root)

		origDir, _ := os.Getwd()
		defer os.Chdir(origDir)
		if err := os.Chdir(dir); err != nil {
			t.Fatal(err)
		}

		output := captureStdout(t, func() {
			rootCmd.SetArgs([]string{"clean"})
			_ = rootCmd.Execute()
		})

		if !strings.Contains(output, "Deleted 0 unused venvs") {
			t.Errorf("expected 'Deleted 0 unused venvs' in output, got: %q", output)
		}
	})

	// global_flag_is_accepted: --global must not produce a flag-parsing error.
	t.Run("global_flag_is_accepted", func(t *testing.T) {
		t.Cleanup(core.ResetConfigCache)

		output := captureStdout(t, func() {
			rootCmd.SetArgs([]string{"clean", "--global"})
			_ = rootCmd.Execute()
		})

		if strings.Contains(output, "unknown flag") {
			t.Errorf("unexpected flag error in output: %q", output)
		}
	})
}
