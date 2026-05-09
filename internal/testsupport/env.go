// Package testsupport provides shared helpers for tests across internal/*
// packages. Although it isn't a `_test` package (so it can be imported from
// any test binary), it is intended for use only from _test.go files.
package testsupport

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	cache "github.com/jlkendrick/grimoire/internal/cache"
	scroll "github.com/jlkendrick/grimoire/internal/scroll"
	utils "github.com/jlkendrick/grimoire/internal/utils"
)

// SetupGrimoireHome redirects $GRIMOIRE_HOME to a per-test temp dir, ensures
// the standard subdirs exist, and resets the package-level scroll/cache memos
// both at setup and via t.Cleanup. Returns the absolute (symlink-resolved)
// home path.
func SetupGrimoireHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	abs, err := filepath.EvalSymlinks(home)
	if err != nil {
		t.Fatalf("EvalSymlinks home: %v", err)
	}
	t.Setenv("GRIMOIRE_HOME", abs)
	if err := utils.EnsureGrimoireSetup(); err != nil {
		t.Fatalf("EnsureGrimoireSetup: %v", err)
	}
	scroll.ResetScrollCache()
	cache.ResetCache()
	t.Cleanup(func() {
		scroll.ResetScrollCache()
		cache.ResetCache()
	})
	return abs
}

// WithScrollDir creates a temp project dir, chdirs into it, and registers a
// cleanup to return to the original cwd. Returns the symlink-resolved
// absolute path (matters on darwin where /tmp -> /private/tmp).
func WithScrollDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	abs, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatalf("EvalSymlinks dir: %v", err)
	}
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(abs); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })
	return abs
}

// WriteFile writes content to path or fails the test.
func WriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// WriteScrollYAML writes body to <dir>/scroll.yaml and returns the path.
func WriteScrollYAML(t *testing.T, dir, body string) string {
	t.Helper()
	p := filepath.Join(dir, "scroll.yaml")
	WriteFile(t, p, body)
	return p
}

// PythonAvailable reports whether python3 is on $PATH.
func PythonAvailable() bool {
	_, err := exec.LookPath("python3")
	return err == nil
}

// GoAvailable reports whether the go toolchain is on $PATH.
func GoAvailable() bool {
	_, err := exec.LookPath("go")
	return err == nil
}
