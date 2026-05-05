package cmd

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	cache "github.com/jlkendrick/grimoire/internal/cache"
	scroll "github.com/jlkendrick/grimoire/internal/scroll"
	utils "github.com/jlkendrick/grimoire/internal/utils"
)

// setupTestEnv redirects $GRIMOIRE_HOME to a temp dir, ensures the
// home/cache/envs subdirs exist, and resets the package-level scroll/cache
// caches. Returns the absolute path to the fake grimoire home.
func setupTestEnv(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("GRIMOIRE_HOME", home)
	if err := utils.EnsureGrimoireSetup(); err != nil {
		t.Fatalf("EnsureGrimoireSetup: %v", err)
	}
	scroll.ResetScrollCache()
	cache.ResetCache()
	t.Cleanup(func() {
		scroll.ResetScrollCache()
		cache.ResetCache()
	})
	return home
}

// withScrollDir creates a temp project dir, chdirs into it, and registers a
// cleanup that returns to the original cwd. Returns the symlink-resolved
// absolute path (matters on darwin where /tmp -> /private/tmp).
func withScrollDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	abs, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatalf("EvalSymlinks: %v", err)
	}
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(abs); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(orig)
	})
	return abs
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// captureOutput runs fn with os.Stdout and os.Stderr redirected into buffers
// and returns (stdout, stderr) captured during the call. The originals are
// always restored before returning.
func captureOutput(t *testing.T, fn func()) (string, string) {
	t.Helper()
	origOut, origErr := os.Stdout, os.Stderr
	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe stdout: %v", err)
	}
	rErr, wErr, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe stderr: %v", err)
	}
	os.Stdout, os.Stderr = wOut, wErr

	var outBuf, errBuf bytes.Buffer
	doneOut := make(chan struct{})
	doneErr := make(chan struct{})
	go func() { _, _ = io.Copy(&outBuf, rOut); close(doneOut) }()
	go func() { _, _ = io.Copy(&errBuf, rErr); close(doneErr) }()

	defer func() {
		os.Stdout, os.Stderr = origOut, origErr
	}()

	fn()

	_ = wOut.Close()
	_ = wErr.Close()
	<-doneOut
	<-doneErr
	return outBuf.String(), errBuf.String()
}

// resetRootCmdState clears any leftover SetArgs state from prior tests.
func resetRootCmdState(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		rootCmd.SetArgs(nil)
	})
}

// pythonAvailable reports whether `python3` is on $PATH.
func pythonAvailable() bool {
	_, err := exec.LookPath("python3")
	return err == nil
}
