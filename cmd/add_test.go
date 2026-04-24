package cmd

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	core "github.com/jlkendrick/grimoire/core"
)

func TestAddCmd(t *testing.T) {
	rootCmd.SetErr(io.Discard)

	t.Run("invalid format prints error", func(t *testing.T) {
		output := captureStdout(t, func() {
			rootCmd.SetArgs([]string{"add", "no_colon_here"})
			_ = rootCmd.Execute()
		})
		if !strings.Contains(output, "path_to_function:function_name format is required") {
			t.Errorf("expected format error in output, got: %q", output)
		}
	})

	t.Run("adds function to existing scroll.yaml", func(t *testing.T) {
		t.Cleanup(core.ResetConfigCache)
		dir := t.TempDir()
		pyContent := "def greet(name: str, times: int = 3):\n    pass\n"
		if err := os.WriteFile(filepath.Join(dir, "greet.py"), []byte(pyContent), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "scroll.yaml"), []byte("{}\n"), 0644); err != nil {
			t.Fatal(err)
		}

		origDir, _ := os.Getwd()
		defer os.Chdir(origDir)
		if err := os.Chdir(dir); err != nil {
			t.Fatal(err)
		}

		output := captureStdout(t, func() {
			rootCmd.SetArgs([]string{"add", "greet.py:greet"})
			_ = rootCmd.Execute()
		})

		if !strings.Contains(output, "Function greet added") {
			t.Errorf("expected success message in output, got: %q", output)
		}

		content, err := os.ReadFile(filepath.Join(dir, "scroll.yaml"))
		if err != nil {
			t.Fatal(err)
		}
		s := string(content)
		for _, want := range []string{"greet", "name", "times"} {
			if !strings.Contains(s, want) {
				t.Errorf("expected %q in updated scroll.yaml, got:\n%s", want, s)
			}
		}
	})

	t.Run("upward traversal uses parent scroll.yaml not CWD", func(t *testing.T) {
		// Verify that when scroll.yaml exists in a parent dir, the add command
		// operates on that file rather than creating a new scroll.yaml in CWD.
		// We intentionally pass a nonexistent Python file so the command fails
		// at GenerateFunctionConfig — but the error message confirms it got past
		// the "no config found" path, proving traversal worked.
		t.Cleanup(core.ResetConfigCache)
		parent := t.TempDir()
		child := filepath.Join(parent, "subdir")
		if err := os.Mkdir(child, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(parent, "scroll.yaml"), []byte("{}\n"), 0644); err != nil {
			t.Fatal(err)
		}

		origDir, _ := os.Getwd()
		defer os.Chdir(origDir)
		if err := os.Chdir(child); err != nil {
			t.Fatal(err)
		}

		output := captureStdout(t, func() {
			rootCmd.SetArgs([]string{"add", "missing.py:func"})
			_ = rootCmd.Execute()
		})

		// The command should attempt to generate the function config (proving it
		// found the existing config) and then fail on the missing file.
		if !strings.Contains(output, "Error generating function config") {
			t.Errorf("expected 'Error generating function config' in output, got: %q", output)
		}
	})

	t.Run("python file not found prints error", func(t *testing.T) {
		t.Cleanup(core.ResetConfigCache)
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "scroll.yaml"), []byte("{}\n"), 0644); err != nil {
			t.Fatal(err)
		}

		origDir, _ := os.Getwd()
		defer os.Chdir(origDir)
		if err := os.Chdir(dir); err != nil {
			t.Fatal(err)
		}

		output := captureStdout(t, func() {
			rootCmd.SetArgs([]string{"add", "nonexistent.py:func"})
			_ = rootCmd.Execute()
		})

		if !strings.Contains(output, "Error generating function config") {
			t.Errorf("expected error in output, got: %q", output)
		}
	})
}
