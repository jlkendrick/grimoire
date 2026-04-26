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

	t.Run("global syncs every registered scroll without corrupting index", func(t *testing.T) {
		t.Cleanup(core.ResetConfigCache)

		// Two registered scrolls, each with one Python function.
		scroll_a_dir := t.TempDir()
		py_a := "def alpha(count: int):\n    pass\n"
		if err := os.WriteFile(filepath.Join(scroll_a_dir, "alpha.py"), []byte(py_a), 0644); err != nil {
			t.Fatal(err)
		}
		scroll_a_path := filepath.Join(scroll_a_dir, "scroll.yaml")
		scroll_a_content := "functions:\n- name: alpha\n  path: alpha.py\n  function: alpha\n"
		if err := os.WriteFile(scroll_a_path, []byte(scroll_a_content), 0644); err != nil {
			t.Fatal(err)
		}

		scroll_b_dir := t.TempDir()
		py_b := "def beta(label: str = \"hi\"):\n    pass\n"
		if err := os.WriteFile(filepath.Join(scroll_b_dir, "beta.py"), []byte(py_b), 0644); err != nil {
			t.Fatal(err)
		}
		scroll_b_path := filepath.Join(scroll_b_dir, "scroll.yaml")
		scroll_b_content := "functions:\n- name: beta\n  path: beta.py\n  function: beta\n"
		if err := os.WriteFile(scroll_b_path, []byte(scroll_b_content), 0644); err != nil {
			t.Fatal(err)
		}

		// Global grimoire pointing at both.
		home := t.TempDir()
		global_path := filepath.Join(home, "grimoire.yaml")
		global_content := "registered_projects:\n- path: " + scroll_a_path + "\n- path: " + scroll_b_path + "\n"
		if err := os.WriteFile(global_path, []byte(global_content), 0644); err != nil {
			t.Fatal(err)
		}
		t.Setenv("GRIMOIRE_HOME", home)

		rootCmd.SetArgs([]string{"sync", "-g"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		updated_a, err := os.ReadFile(scroll_a_path)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(updated_a), "name: count") {
			t.Errorf("expected scroll A to gain arg 'count', got:\n%s", string(updated_a))
		}

		updated_b, err := os.ReadFile(scroll_b_path)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(updated_b), "name: label") {
			t.Errorf("expected scroll B to gain arg 'label', got:\n%s", string(updated_b))
		}

		// The global file must remain a pure index — no inlined functions.
		updated_global, err := os.ReadFile(global_path)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(updated_global), "functions:") {
			t.Errorf("global grimoire should not contain inlined functions, got:\n%s", string(updated_global))
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
