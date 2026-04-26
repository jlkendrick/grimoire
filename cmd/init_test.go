package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeTempFile creates a temporary file with the given content and returns its
// path along with a cleanup function.
func writeTempFile(t *testing.T, pattern, content string) (path string, cleanup func()) {
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
	rootCmd.SetErr(io.Discard)

	t.Run("creates scroll.yaml in current directory", func(t *testing.T) {
		dir := t.TempDir()
		origDir, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Chdir(dir); err != nil {
			t.Fatal(err)
		}
		defer os.Chdir(origDir)

		rootCmd.SetArgs([]string{"init"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if _, err := os.Stat(filepath.Join(dir, "scroll.yaml")); os.IsNotExist(err) {
			t.Error("expected scroll.yaml to be created in current directory")
		}
	})

	t.Run("created file contains boilerplate", func(t *testing.T) {
		dir := t.TempDir()
		origDir, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Chdir(dir); err != nil {
			t.Fatal(err)
		}
		defer os.Chdir(origDir)

		rootCmd.SetArgs([]string{"init"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, err := os.ReadFile(filepath.Join(dir, "scroll.yaml"))
		if err != nil {
			t.Fatalf("reading scroll.yaml: %v", err)
		}
		s := string(content)
		for _, want := range []string{"functions:", "hello_world"} {
			if !strings.Contains(s, want) {
				t.Errorf("expected %q in file, got:\n%s", want, s)
			}
		}
	})
}
