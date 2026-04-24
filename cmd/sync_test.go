package cmd

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	core "github.com/jlkendrick/grimoire/core"
)

func TestSyncCmd(t *testing.T) {
	rootCmd.SetErr(io.Discard)

	t.Run("updates args from source files", func(t *testing.T) {
		t.Cleanup(core.ResetConfigCache)
		dir := t.TempDir()
		pyContent := "def greet(name: str, times: int = 3):\n    pass\n"
		pyPath := filepath.Join(dir, "greet.py")
		if err := os.WriteFile(pyPath, []byte(pyContent), 0644); err != nil {
			t.Fatal(err)
		}

		// Use absolute path in the config so ExtractSignature can find the file
		// regardless of CWD.
		grimContent := "functions:\n- name: greet\n  path: " + pyPath + "\n  function: greet\n"
		grimPath := filepath.Join(dir, "scroll.yaml")
		if err := os.WriteFile(grimPath, []byte(grimContent), 0644); err != nil {
			t.Fatal(err)
		}

		origDir, _ := os.Getwd()
		defer os.Chdir(origDir)
		if err := os.Chdir(dir); err != nil {
			t.Fatal(err)
		}

		rootCmd.SetArgs([]string{"sync"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, err := os.ReadFile(grimPath)
		if err != nil {
			t.Fatal(err)
		}
		s := string(content)
		for _, want := range []string{"name: name", "name: times"} {
			if !strings.Contains(s, want) {
				t.Errorf("expected %q in synced scroll.yaml, got:\n%s", want, s)
			}
		}
	})

	t.Run("function with invalid path prints error", func(t *testing.T) {
		t.Cleanup(core.ResetConfigCache)
		dir := t.TempDir()
		grimContent := "functions:\n- name: greet\n  path: /nonexistent/grimoire/path/greet.py\n  function: greet\n"
		grimPath := filepath.Join(dir, "scroll.yaml")
		if err := os.WriteFile(grimPath, []byte(grimContent), 0644); err != nil {
			t.Fatal(err)
		}

		origDir, _ := os.Getwd()
		defer os.Chdir(origDir)
		if err := os.Chdir(dir); err != nil {
			t.Fatal(err)
		}

		output := captureStdout(t, func() {
			rootCmd.SetArgs([]string{"sync"})
			_ = rootCmd.Execute()
		})

		if !strings.Contains(output, "Error generating function config") {
			t.Errorf("expected error in output, got: %q", output)
		}
	})

	t.Run("empty functions list writes file without error", func(t *testing.T) {
		t.Cleanup(core.ResetConfigCache)
		dir := t.TempDir()
		grimPath := filepath.Join(dir, "scroll.yaml")
		if err := os.WriteFile(grimPath, []byte("{}\n"), 0644); err != nil {
			t.Fatal(err)
		}

		origDir, _ := os.Getwd()
		defer os.Chdir(origDir)
		if err := os.Chdir(dir); err != nil {
			t.Fatal(err)
		}

		rootCmd.SetArgs([]string{"sync"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
