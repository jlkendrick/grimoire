package runtimes

import (
	"os"
	"errors"
	"strings"
	"testing"
	"path/filepath"

	utils "github.com/jlkendrick/grimoire/utils"
)

// -------------------------------------------------------------------------
func TestPythonAdapter_FormatError(t *testing.T) {
	adapter := &PythonAdapter{}

	t.Run("wraps_error_with_prefix", func(t *testing.T) {
		wrapped := adapter.FormatError(errors.New("something went wrong"))
		if wrapped.Error() != "python runtime error: something went wrong" {
			t.Errorf("unexpected error string: %q", wrapped.Error())
		}
	})

	t.Run("preserves_original_message", func(t *testing.T) {
		wrapped := adapter.FormatError(errors.New("exit status 1"))
		if !strings.HasPrefix(wrapped.Error(), "python runtime error:") {
			t.Errorf("missing prefix in: %q", wrapped.Error())
		}
		if !strings.Contains(wrapped.Error(), "exit status 1") {
			t.Errorf("original message not preserved in: %q", wrapped.Error())
		}
	})
}

// -------------------------------------------------------------------------
// TestUpwardsTraversalForTargets
// -------------------------------------------------------------------------

func TestUpwardsTraversalForTargets(t *testing.T) {
	targets := []string{".venv", "pyproject.toml", "requirements.txt"}

	t.Run("venv_only", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, ".venv"), 0755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}

		matched, found := utils.UpwardsTraversalForTargets(dir, targets)
		if !found {
			t.Fatal("expected found=true, got false")
		}
		if matched[".venv"] == "" {
			t.Error("expected .venv to be set")
		}
		if v, ok := matched["pyproject.toml"]; ok {
			t.Errorf("expected pyproject.toml to be absent, got %q", v)
		}
		if v, ok := matched["requirements.txt"]; ok {
			t.Errorf("expected requirements.txt to be absent, got %q", v)
		}
	})

	t.Run("pyproject_only", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte("[project]\n"), 0644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}

		matched, found := utils.UpwardsTraversalForTargets(dir, targets)
		if !found {
			t.Fatal("expected found=true, got false")
		}
		if v, ok := matched[".venv"]; ok {
			t.Errorf("expected .venv to be absent, got %q", v)
		}
		if matched["pyproject.toml"] == "" {
			t.Error("expected pyproject.toml to be set")
		}
		if v, ok := matched["requirements.txt"]; ok {
			t.Errorf("expected requirements.txt to be absent, got %q", v)
		}
	})

	t.Run("requirements_only", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte(""), 0644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}

		matched, found := utils.UpwardsTraversalForTargets(dir, targets)
		if !found {
			t.Fatal("expected found=true, got false")
		}
		if v, ok := matched[".venv"]; ok {
			t.Errorf("expected .venv to be absent, got %q", v)
		}
		if v, ok := matched["pyproject.toml"]; ok {
			t.Errorf("expected pyproject.toml to be absent, got %q", v)
		}
		if matched["requirements.txt"] == "" {
			t.Error("expected requirements.txt to be set")
		}
	})

	t.Run("all_three_present", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, ".venv"), 0755); err != nil {
			t.Fatalf("MkdirAll .venv: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte("[project]\n"), 0644); err != nil {
			t.Fatalf("WriteFile pyproject.toml: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte(""), 0644); err != nil {
			t.Fatalf("WriteFile requirements.txt: %v", err)
		}

		matched, found := utils.UpwardsTraversalForTargets(dir, targets)
		if !found {
			t.Fatal("expected found=true, got false")
		}
		if matched[".venv"] == "" {
			t.Error("expected .venv to be set")
		}
		if matched["pyproject.toml"] == "" {
			t.Error("expected pyproject.toml to be set")
		}
		if matched["requirements.txt"] == "" {
			t.Error("expected requirements.txt to be set")
		}
	})

	t.Run("recurse_to_parent", func(t *testing.T) {
		parent := t.TempDir()
		child := filepath.Join(parent, "child")
		if err := os.MkdirAll(child, 0755); err != nil {
			t.Fatalf("MkdirAll child: %v", err)
		}
		// Only the parent has .venv; child has nothing.
		if err := os.MkdirAll(filepath.Join(parent, ".venv"), 0755); err != nil {
			t.Fatalf("MkdirAll .venv: %v", err)
		}

		matched, found := utils.UpwardsTraversalForTargets(child, targets)
		if !found {
			t.Fatal("expected found=true after traversing to parent, got false")
		}
		venvPath := matched[".venv"]
		if venvPath == "" {
			t.Fatal("expected .venv to be set after traversing to parent")
		}
		if !strings.HasSuffix(venvPath, ".venv") {
			t.Errorf("expected venvPath to end with .venv, got %q", venvPath)
		}
	})

	t.Run("root_not_found", func(t *testing.T) {
		// Use a fresh temp dir with nothing in it. The traversal will hit
		// system directories (none of which have .venv/pyproject/requirements)
		// and eventually reach the filesystem root.
		dir := t.TempDir()

		_, found := utils.UpwardsTraversalForTargets(dir, targets)
		if found {
			t.Fatal("expected found=false for directory with no env markers, got true")
		}
	})
}

// -------------------------------------------------------------------------
func TestBuildNewEnvironment(t *testing.T) {
	t.Run("creates_venv_from_requirements", func(t *testing.T) {
		requirePython(t)

		workDir := t.TempDir()
		reqFile := filepath.Join(workDir, "requirements.txt")
		if err := os.WriteFile(reqFile, []byte(""), 0644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}

		got, _, err := buildNewEnvironment(reqFile, "requirements.txt", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.HasSuffix(got, filepath.Join("bin", "python")) {
			t.Errorf("expected path ending in bin/python, got %q", got)
		}

		// The returned python binary should exist.
		if _, err := os.Stat(got); err != nil {
			t.Errorf("returned python path %q does not exist: %v", got, err)
		}

		// The .grimoire_req_hash certificate should have been written.
		venvDir := filepath.Dir(filepath.Dir(got)) // strip "bin/python" → venv root
		hashFile := filepath.Join(venvDir, ".grimoire_req_hash")
		if _, err := os.Stat(hashFile); err != nil {
			t.Errorf(".grimoire_req_hash not found at %q: %v", hashFile, err)
		}
	})

	t.Run("reuses_venv_on_same_hash", func(t *testing.T) {
		requirePython(t)

		workDir := t.TempDir()
		reqFile := filepath.Join(workDir, "requirements.txt")
		if err := os.WriteFile(reqFile, []byte(""), 0644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}

		first, _, err := buildNewEnvironment(reqFile, "requirements.txt", "")
		if err != nil {
			t.Fatalf("first call unexpected error: %v", err)
		}

		second, _, err := buildNewEnvironment(reqFile, "requirements.txt", "")
		if err != nil {
			t.Fatalf("second call unexpected error: %v", err)
		}

		if first != second {
			t.Errorf("expected same path on second call\n  first:  %q\n  second: %q", first, second)
		}
	})

	t.Run("recreates_on_hash_mismatch", func(t *testing.T) {
		requirePython(t)

		workDir := t.TempDir()
		reqFile := filepath.Join(workDir, "requirements.txt")
		if err := os.WriteFile(reqFile, []byte(""), 0644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}

		// First build.
		_, _, err := buildNewEnvironment(reqFile, "requirements.txt", "")
		if err != nil {
			t.Fatalf("first call unexpected error: %v", err)
		}

		// Change the requirements file content to trigger a hash mismatch.
		if err := os.WriteFile(reqFile, []byte("# changed\n"), 0644); err != nil {
			t.Fatalf("WriteFile (update): %v", err)
		}

		// Second build should succeed and update the hash certificate.
		got, _, err := buildNewEnvironment(reqFile, "requirements.txt", "")
		if err != nil {
			t.Fatalf("second call unexpected error: %v", err)
		}

		// Verify the hash certificate was updated.
		venvDir := filepath.Dir(filepath.Dir(got))
		hashFile := filepath.Join(venvDir, ".grimoire_req_hash")
		updated, err := os.ReadFile(hashFile)
		if err != nil {
			t.Fatalf("reading updated hash file: %v", err)
		}
		// The stored hash should be non-empty (exact value not asserted, just that it was written).
		if len(updated) == 0 {
			t.Error("expected non-empty hash in .grimoire_req_hash after update")
		}
	})

	t.Run("unsupported_dependency_type_returns_error", func(t *testing.T) {
		requirePython(t)

		workDir := t.TempDir()
		reqFile := filepath.Join(workDir, "Pipfile")
		if err := os.WriteFile(reqFile, []byte(""), 0644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}

		_, _, err := buildNewEnvironment(reqFile, "Pipfile", "")
		if err == nil {
			t.Fatal("expected error for unsupported dependency type, got nil")
		}
		if !strings.Contains(err.Error(), "unsupported dependency type") {
			t.Errorf("expected 'unsupported dependency type' in error, got: %q", err.Error())
		}
	})

	t.Run("writes_origin_file_with_supplied_path", func(t *testing.T) {
		requirePython(t)

		workDir := t.TempDir()
		reqFile := filepath.Join(workDir, "requirements.txt")
		if err := os.WriteFile(reqFile, []byte(""), 0644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}

		wantOrigin := "/abs/path/to/my_script.py"
		got, _, err := buildNewEnvironment(reqFile, "requirements.txt", wantOrigin)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		venvDir := filepath.Dir(filepath.Dir(got)) // strip bin/python → venv root
		originFile := filepath.Join(venvDir, ".grimoire_origin")
		content, err := os.ReadFile(originFile)
		if err != nil {
			t.Fatalf(".grimoire_origin not found at %q: %v", originFile, err)
		}
		if string(content) != wantOrigin {
			t.Errorf("origin = %q, want %q", string(content), wantOrigin)
		}
	})

	t.Run("recovers_from_missing_hash_file", func(t *testing.T) {
		requirePython(t)

		workDir := t.TempDir()
		reqFile := filepath.Join(workDir, "requirements.txt")
		if err := os.WriteFile(reqFile, []byte(""), 0644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}

		// First build — creates venv and writes hash certificate.
		got, _, err := buildNewEnvironment(reqFile, "requirements.txt", "")
		if err != nil {
			t.Fatalf("first call unexpected error: %v", err)
		}

		// Manually delete the hash certificate to simulate the bug scenario.
		venvDir := filepath.Dir(filepath.Dir(got))
		hashFile := filepath.Join(venvDir, ".grimoire_req_hash")
		if err := os.Remove(hashFile); err != nil {
			t.Fatalf("removing hash file: %v", err)
		}

		// Second call should recover cleanly rather than erroring.
		_, _, err = buildNewEnvironment(reqFile, "requirements.txt", "")
		if err != nil {
			t.Errorf("expected clean recovery from missing hash file, got: %v", err)
		}
	})
}
