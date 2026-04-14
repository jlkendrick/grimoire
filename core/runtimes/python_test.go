package runtimes

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	types "github.com/jlkendrick/grimoire/types"
	utils "github.com/jlkendrick/grimoire/utils"
)

// requirePython skips the test if the "python" binary is not on PATH.
func requirePython(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("python"); err != nil {
		t.Skip("python binary not found on PATH; skipping integration test")
	}
}

// -------------------------------------------------------------------------
// TestPythonAdapter_GenerateCommand
//
// GenerateCommand now calls GetInterpreter internally. Each test case sets
// Interpreter explicitly so GetInterpreter returns via Tier 1 (explicit
// path) without touching the filesystem.
// -------------------------------------------------------------------------

func TestPythonAdapter_GenerateCommand(t *testing.T) {
	tests := []struct {
		name           string
		targetFile     string
		targetFunction string
		wantTargetDir  string
		wantModule     string
	}{
		{
			name:           "simple_relative_path",
			targetFile:     "sample/hello_world_func.py",
			targetFunction: "hello_world",
			wantTargetDir:  "sample",
			wantModule:     "hello_world_func",
		},
		{
			name:           "nested_path",
			targetFile:     "a/b/c/my_func.py",
			targetFunction: "run",
			wantTargetDir:  "a/b/c",
			wantModule:     "my_func",
		},
		{
			name:           "absolute_path",
			targetFile:     "/home/user/scripts/processor.py",
			targetFunction: "process",
			wantTargetDir:  "/home/user/scripts",
			wantModule:     "processor",
		},
		{
			name:           "flat_no_directory",
			targetFile:     "flat.py",
			targetFunction: "main",
			wantTargetDir:  ".",
			wantModule:     "flat",
		},
		{
			// Documents existing behavior: TrimSuffix removes only ".py",
			// leaving dots in the stem. importlib would fail at runtime on
			// such a path, but GenerateCommand itself does not error.
			name:           "file_with_multiple_dots",
			targetFile:     "src/my.util.helper.py",
			targetFunction: "compute",
			wantTargetDir:  "src",
			wantModule:     "my.util.helper",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			adapter := &PythonAdapter{}
			fn := types.Function{
				TargetFile:     tc.targetFile,
				TargetFunction: tc.targetFunction,
				// Set explicit interpreter so GetInterpreter short-circuits
				// via Tier 1 and does not traverse the filesystem.
				Interpreter: "python",
			}

			binary, flags, err := adapter.GenerateCommand(fn)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if binary != "python" {
				t.Errorf("expected binary %q, got %q", "python", binary)
			}
			if len(flags) != 2 {
				t.Fatalf("expected 2 flags, got %d: %v", len(flags), flags)
			}
			if flags[0] != "-c" {
				t.Errorf("expected flags[0]==-c, got %q", flags[0])
			}

			script := flags[1]

			// Shared structural assertions present in every generated script.
			for _, want := range []string{
				"importlib.import_module",
				"sys.stdin.read",
				"getattr",
			} {
				if !strings.Contains(script, want) {
					t.Errorf("script missing %q", want)
				}
			}

			// Verify the three fmt.Sprintf substitutions.
			wantDirLiteral := "os.path.expanduser('" + tc.wantTargetDir + "')"
			if !strings.Contains(script, wantDirLiteral) {
				t.Errorf("script missing target_dir literal %q\nscript:\n%s", wantDirLiteral, script)
			}

			wantModLiteral := "importlib.import_module('" + tc.wantModule + "')"
			if !strings.Contains(script, wantModLiteral) {
				t.Errorf("script missing module literal %q\nscript:\n%s", wantModLiteral, script)
			}

			wantFnLiteral := "getattr(mod, '" + tc.targetFunction + "')"
			if !strings.Contains(script, wantFnLiteral) {
				t.Errorf("script missing function literal %q\nscript:\n%s", wantFnLiteral, script)
			}
		})
	}
}

// -------------------------------------------------------------------------
// TestPythonAdapter_FormatError
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
// TestGetInterpreter
// -------------------------------------------------------------------------

func TestGetInterpreter(t *testing.T) {
	adapter := &PythonAdapter{}

	t.Run("explicit_interpreter", func(t *testing.T) {
		fn := types.Function{
			TargetFile:  "sample/script.py",
			Interpreter: "/usr/bin/python3",
		}
		got, err := adapter.GetInterpreter(fn)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "/usr/bin/python3" {
			t.Errorf("expected %q, got %q", "/usr/bin/python3", got)
		}
	})

	t.Run("explicit_interpreter_tilde", func(t *testing.T) {
		home, err := os.UserHomeDir()
		if err != nil {
			t.Fatalf("UserHomeDir: %v", err)
		}
		fn := types.Function{
			TargetFile:  "sample/script.py",
			Interpreter: "~/venv/bin/python",
		}
		got, err := adapter.GetInterpreter(fn)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := filepath.Join(home, "venv", "bin", "python")
		if got != want {
			t.Errorf("expected %q, got %q", want, got)
		}
	})

	t.Run("venv_in_same_dir", func(t *testing.T) {
		dir := t.TempDir()
		// Create a .venv directory (just needs to exist, not be a real venv).
		if err := os.MkdirAll(filepath.Join(dir, ".venv", "bin"), 0755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}

		fn := types.Function{
			// TargetFile must be an absolute path in the temp dir.
			TargetFile:     filepath.Join(dir, "script.py"),
			TargetFunction: "run",
		}
		got, err := adapter.GetInterpreter(fn)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := filepath.Join(dir, ".venv", "bin", "python")
		if got != want {
			t.Errorf("expected %q, got %q", want, got)
		}
	})

	t.Run("venv_in_parent_dir", func(t *testing.T) {
		parent := t.TempDir()
		child := filepath.Join(parent, "subpkg")
		if err := os.MkdirAll(child, 0755); err != nil {
			t.Fatalf("MkdirAll child: %v", err)
		}
		// Only the parent has .venv; the child has nothing.
		if err := os.MkdirAll(filepath.Join(parent, ".venv", "bin"), 0755); err != nil {
			t.Fatalf("MkdirAll .venv: %v", err)
		}

		fn := types.Function{
			TargetFile:     filepath.Join(child, "script.py"),
			TargetFunction: "run",
		}
		got, err := adapter.GetInterpreter(fn)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := filepath.Join(parent, ".venv", "bin", "python")
		if got != want {
			t.Errorf("expected %q, got %q", want, got)
		}
	})

	t.Run("no_env_found_falls_back_to_system_python", func(t *testing.T) {
		// When no env markers exist anywhere in the directory tree,
		// GetInterpreter now falls back to the system "python" binary.
		dir := t.TempDir()
		fn := types.Function{
			TargetFile:     filepath.Join(dir, "script.py"),
			TargetFunction: "run",
		}
		got, err := adapter.GetInterpreter(fn)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "python" {
			t.Errorf("expected fallback to %q, got %q", "python", got)
		}
	})
}

// -------------------------------------------------------------------------
// TestBuildNewEnvironment — integration tests, require python + pip
// -------------------------------------------------------------------------

func TestBuildNewEnvironment(t *testing.T) {
	t.Run("creates_venv_from_requirements", func(t *testing.T) {
		requirePython(t)

		workDir := t.TempDir()
		reqFile := filepath.Join(workDir, "requirements.txt")
		if err := os.WriteFile(reqFile, []byte(""), 0644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}

		got, err := buildNewEnvironment(reqFile, "requirements.txt", "")
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

		first, err := buildNewEnvironment(reqFile, "requirements.txt", "")
		if err != nil {
			t.Fatalf("first call unexpected error: %v", err)
		}

		second, err := buildNewEnvironment(reqFile, "requirements.txt", "")
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
		_, err := buildNewEnvironment(reqFile, "requirements.txt", "")
		if err != nil {
			t.Fatalf("first call unexpected error: %v", err)
		}

		// Change the requirements file content to trigger a hash mismatch.
		if err := os.WriteFile(reqFile, []byte("# changed\n"), 0644); err != nil {
			t.Fatalf("WriteFile (update): %v", err)
		}

		// Second build should succeed and update the hash certificate.
		got, err := buildNewEnvironment(reqFile, "requirements.txt", "")
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

		_, err := buildNewEnvironment(reqFile, "Pipfile", "")
		if err == nil {
			t.Fatal("expected error for unsupported dependency type, got nil")
		}
		if !strings.Contains(err.Error(), "unsupported dependency type") {
			t.Errorf("expected 'unsupported dependency type' in error, got: %q", err.Error())
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
		got, err := buildNewEnvironment(reqFile, "requirements.txt", "")
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
		_, err = buildNewEnvironment(reqFile, "requirements.txt", "")
		if err != nil {
			t.Errorf("expected clean recovery from missing hash file, got: %v", err)
		}
	})
}
