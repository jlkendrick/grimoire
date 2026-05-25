package scroll_test

import (
	"path/filepath"
	"strings"
	"testing"

	scroll "github.com/jlkendrick/grimoire/internal/scroll"
	ts "github.com/jlkendrick/grimoire/internal/testsupport"
)

func TestParseScroll_NameDefaultsFromDir(t *testing.T) {
	ts.SetupGrimoireHome(t)
	dir := ts.WithScrollDir(t)
	path := ts.WriteScrollYAML(t, dir, "spells:\n  - command: c\n    path: c.py\n    function: c\n")

	s, err := scroll.ParseScroll(path)
	if err != nil {
		t.Fatalf("ParseScroll: %v", err)
	}
	want := filepath.Base(filepath.Dir(path))
	if s.Name != want {
		t.Errorf("Name = %q, want %q", s.Name, want)
	}
}

func TestParseScroll_ExplicitNameKept(t *testing.T) {
	ts.SetupGrimoireHome(t)
	dir := ts.WithScrollDir(t)
	path := ts.WriteScrollYAML(t, dir, "name: myproj\nspells:\n  - command: c\n    path: c.py\n    function: c\n")

	s, err := scroll.ParseScroll(path)
	if err != nil {
		t.Fatalf("ParseScroll: %v", err)
	}
	if s.Name != "myproj" {
		t.Errorf("Name = %q, want %q", s.Name, "myproj")
	}
}

func TestParseScroll_RegistryHasNoName(t *testing.T) {
	ts.SetupGrimoireHome(t)
	dir := ts.WithScrollDir(t)
	path := ts.WriteScrollYAML(t, dir, "registered_scrolls:\n  - path: /some/path/scroll.yaml\n")

	s, err := scroll.ParseScroll(path)
	if err != nil {
		t.Fatalf("ParseScroll: %v", err)
	}
	if s.Name != "" {
		t.Errorf("registry Name = %q, want empty (no spells/rituals -> no synthetic name)", s.Name)
	}
}

func TestParseScroll_RejectsExplicitEmptyName(t *testing.T) {
	ts.SetupGrimoireHome(t)
	dir := ts.WithScrollDir(t)
	path := ts.WriteScrollYAML(t, dir, "name: \nspells:\n  - command: c\n    path: c.py\n    function: c\n")

	_, err := scroll.ParseScroll(path)
	if err == nil {
		t.Fatalf("expected error for explicit empty name, got nil")
	}
	if !strings.Contains(err.Error(), "empty `name:`") {
		t.Errorf("error = %v, want one mentioning explicit empty name", err)
	}
}

func TestParseScroll_RejectsInvalidNameChars(t *testing.T) {
	cases := []struct {
		label, name string
	}{
		{"dot", "with.dot"},
		{"space", "two words"},
		{"slash", "a/b"},
	}
	for _, c := range cases {
		t.Run(c.label, func(t *testing.T) {
			ts.SetupGrimoireHome(t)
			dir := ts.WithScrollDir(t)
			body := "name: \"" + c.name + "\"\nspells:\n  - command: cmd\n    path: c.py\n    function: c\n"
			path := ts.WriteScrollYAML(t, dir, body)

			_, err := scroll.ParseScroll(path)
			if err == nil {
				t.Fatalf("expected error for name %q, got nil", c.name)
			}
			if !strings.Contains(err.Error(), "invalid name") {
				t.Errorf("error = %v, want one mentioning invalid name", err)
			}
		})
	}
}

func TestParseScroll_RejectsDuplicateSpellCommand(t *testing.T) {
	ts.SetupGrimoireHome(t)
	dir := ts.WithScrollDir(t)
	path := ts.WriteScrollYAML(t, dir, "spells:\n  - command: dup\n    path: a.py\n    function: a\n  - command: dup\n    path: b.py\n    function: b\n")

	_, err := scroll.ParseScroll(path)
	if err == nil {
		t.Fatalf("expected error for duplicate spell command, got nil")
	}
	if !strings.Contains(err.Error(), "declared more than once") {
		t.Errorf("error = %v, want duplicate-command error", err)
	}
}

func TestParseScroll_RejectsSpellRitualCommandCollision(t *testing.T) {
	ts.SetupGrimoireHome(t)
	dir := ts.WithScrollDir(t)
	body := "spells:\n  - command: shared\n    path: a.py\n    function: a\nrituals:\n  - command: shared\n    steps:\n      - spell: shared\n"
	path := ts.WriteScrollYAML(t, dir, body)

	_, err := scroll.ParseScroll(path)
	if err == nil {
		t.Fatalf("expected error for spell/ritual command collision, got nil")
	}
	if !strings.Contains(err.Error(), "declared more than once") {
		t.Errorf("error = %v, want duplicate-command error", err)
	}
}
