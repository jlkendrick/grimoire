package core

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitScroll(t *testing.T) {
	t.Run("with boilerplate creates expected content", func(t *testing.T) {
		dir := t.TempDir()
		cfg, err := InitScroll(dir, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Path != filepath.Join(dir, "scroll.yaml") {
			t.Errorf("unexpected cfg.Path: %s", cfg.Path)
		}
		content, err := os.ReadFile(cfg.Path)
		if err != nil {
			t.Fatalf("reading scroll.yaml: %v", err)
		}
		s := string(content)
		for _, want := range []string{"functions:", "hello_world", "path/to/hello_world.py"} {
			if !strings.Contains(s, want) {
				t.Errorf("expected %q in output, got:\n%s", want, s)
			}
		}
	})

	t.Run("without boilerplate omits example function", func(t *testing.T) {
		dir := t.TempDir()
		cfg, err := InitScroll(dir, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		content, err := os.ReadFile(cfg.Path)
		if err != nil {
			t.Fatalf("reading scroll.yaml: %v", err)
		}
		if strings.Contains(string(content), "hello_world") {
			t.Errorf("expected no boilerplate content, got:\n%s", string(content))
		}
	})

	t.Run("returns ErrScrollExists when scroll already exists", func(t *testing.T) {
		dir := t.TempDir()
		if _, err := InitScroll(dir, false); err != nil {
			t.Fatalf("first init failed: %v", err)
		}
		_, err := InitScroll(dir, false)
		if !errors.Is(err, ErrScrollExists) {
			t.Errorf("expected ErrScrollExists, got %v", err)
		}
	})

	t.Run("nonexistent directory returns error", func(t *testing.T) {
		_, err := InitScroll("/nonexistent/path/that/does/not/exist", false)
		if err == nil {
			t.Error("expected error for nonexistent directory, got nil")
		}
	})
}

func TestRegisterScroll(t *testing.T) {
	t.Run("appends scroll path to global config", func(t *testing.T) {
		t.Cleanup(ResetConfigCache)
		home := t.TempDir()
		grimoire_path := filepath.Join(home, "grimoire.yaml")
		if err := os.WriteFile(grimoire_path, []byte("{}\n"), 0644); err != nil {
			t.Fatal(err)
		}
		t.Setenv("GRIMOIRE_HOME", home)

		scroll_path := filepath.Join(t.TempDir(), "scroll.yaml")
		if err := RegisterScroll(scroll_path); err != nil {
			t.Fatalf("RegisterScroll: %v", err)
		}

		updated, err := os.ReadFile(grimoire_path)
		if err != nil {
			t.Fatalf("reading global config: %v", err)
		}
		if !strings.Contains(string(updated), scroll_path) {
			t.Errorf("expected %q in global config, got:\n%s", scroll_path, string(updated))
		}
	})
}

func TestFindLocalScroll(t *testing.T) {
	t.Run("finds scroll.yaml in current directory", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "scroll.yaml")
		if err := os.WriteFile(path, []byte("{}\n"), 0644); err != nil {
			t.Fatal(err)
		}
		got, found := FindLocalScroll(dir)
		if !found {
			t.Fatal("expected found=true")
		}
		if got != path {
			t.Errorf("expected %q, got %q", path, got)
		}
	})

	t.Run("walks upward to find scroll.yaml", func(t *testing.T) {
		parent := t.TempDir()
		path := filepath.Join(parent, "scroll.yaml")
		if err := os.WriteFile(path, []byte("{}\n"), 0644); err != nil {
			t.Fatal(err)
		}
		child := filepath.Join(parent, "subdir")
		if err := os.Mkdir(child, 0755); err != nil {
			t.Fatal(err)
		}
		got, found := FindLocalScroll(child)
		if !found {
			t.Fatal("expected found=true")
		}
		if got != path {
			t.Errorf("expected %q, got %q", path, got)
		}
	})

	t.Run("returns false when no scroll.yaml exists", func(t *testing.T) {
		dir := t.TempDir()
		_, found := FindLocalScroll(dir)
		if found {
			t.Error("expected found=false")
		}
	})
}
