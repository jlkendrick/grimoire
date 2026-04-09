package cmd

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

// writeTempInitFile creates a temporary file with the given content and returns
// its path along with a cleanup function.
func writeTempInitFile(t *testing.T, pattern, content string) (path string, cleanup func()) {
	t.Helper()
	f, err := os.CreateTemp("", pattern)
	if err != nil {
		t.Fatalf("CreateTemp(%q): %v", pattern, err)
	}
	if _, err := f.WriteString(content); err != nil {
		f.Close()
		os.Remove(f.Name())
		t.Fatalf("WriteString: %v", err)
	}
	f.Close()
	return f.Name(), func() { os.Remove(f.Name()) }
}

// captureStdout runs fn and returns everything written to os.Stdout during that call.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	old := os.Stdout
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("reading captured stdout: %v", err)
	}
	return buf.String()
}

func TestInitCmd(t *testing.T) {
	// initCmd.Execute() traverses to rootCmd in cobra, so we drive tests via
	// rootCmd.SetArgs to ensure the "init" subcommand actually runs.
	// We discard cobra's own error stream to keep test output clean.
	rootCmd.SetErr(io.Discard)

	t.Run("valid config writes manifest with extracted args", func(t *testing.T) {
		pyPath, pyCleanup := writeTempInitFile(t, "test_*.py",
			"def greet(name: str, times: int = 3):\n    pass\n")
		defer pyCleanup()

		cfgContent := "functions:\n- name: greet\n  path: " + pyPath + "\n  function: greet\n"
		cfgPath, cfgCleanup := writeTempInitFile(t, "test_config_*.yaml", cfgContent)
		defer cfgCleanup()

		rootCmd.SetArgs([]string{"init", cfgPath})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		written, err := os.ReadFile(cfgPath)
		if err != nil {
			t.Fatalf("reading written manifest: %v", err)
		}
		content := string(written)
		for _, want := range []string{"name: name", "name: times"} {
			if !strings.Contains(content, want) {
				t.Errorf("manifest missing %q; got:\n%s", want, content)
			}
		}
	})

	t.Run("prints error for missing config file", func(t *testing.T) {
		rootCmd.SetArgs([]string{"init", "/tmp/nonexistent_sigil_test_config.yaml"})
		output := captureStdout(t, func() {
			_ = rootCmd.Execute()
		})
		if !strings.Contains(output, "Error parsing config file") {
			t.Errorf("expected 'Error parsing config file' in stdout, got: %q", output)
		}
	})

	t.Run("requires exactly one argument", func(t *testing.T) {
		rootCmd.SetArgs([]string{"init"})
		err := rootCmd.Execute()
		if err == nil {
			t.Fatal("expected error for zero args, got nil")
		}
	})

	t.Run("prints error for unsupported file extension", func(t *testing.T) {
		rbPath, rbCleanup := writeTempInitFile(t, "test_*.rb", "# ruby\n")
		defer rbCleanup()

		cfgContent := "functions:\n- name: run\n  path: " + rbPath + "\n  function: some_func\n"
		cfgPath, cfgCleanup := writeTempInitFile(t, "test_config_*.yaml", cfgContent)
		defer cfgCleanup()

		rootCmd.SetArgs([]string{"init", cfgPath})
		output := captureStdout(t, func() {
			_ = rootCmd.Execute()
		})
		if !strings.Contains(output, "Error generating manifest YAML") {
			t.Errorf("expected 'Error generating manifest YAML' in stdout, got: %q", output)
		}
	})

	t.Run("raw script with no target function is written as-is", func(t *testing.T) {
		pyPath, pyCleanup := writeTempInitFile(t, "test_*.py", "print('hello')\n")
		defer pyCleanup()

		cfgContent := "functions:\n- name: script\n  path: " + pyPath + "\n"
		cfgPath, cfgCleanup := writeTempInitFile(t, "test_config_*.yaml", cfgContent)
		defer cfgCleanup()

		rootCmd.SetArgs([]string{"init", cfgPath})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		written, err := os.ReadFile(cfgPath)
		if err != nil {
			t.Fatalf("reading written manifest: %v", err)
		}
		// No args section should appear since there was no target function.
		if strings.Contains(string(written), "args:") {
			t.Errorf("expected no 'args:' section for raw script, got:\n%s", string(written))
		}
	})
}
