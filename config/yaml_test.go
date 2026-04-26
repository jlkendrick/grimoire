package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	types "github.com/jlkendrick/grimoire/types"
)

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

func TestParseConfig(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		content := `functions:
- name: greet
  path: test/hello_world_func.py
  function: hello_world
  args:
  - name: "n"
    type: int
    default: "1"
`
		path, cleanup := writeTempFile(t, "test_config_*.yaml", content)
		defer cleanup()

		got, err := ParseConfig(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		want := types.Config{
			Functions: []types.Function{
				{
					Name:           "greet",
					TargetFile:     "test/hello_world_func.py",
					TargetFunction: "hello_world",
					Args: []types.Arg{
						{Name: "n", Type: "int", Default: "1"},
					},
					ScrollPath: path,
				},
			},
			Context: types.ContextTypeLocal,
			Path: path,
		}

		if !reflect.DeepEqual(*got, want) {
			t.Errorf("config mismatch\n  got:  %#v\n  want: %#v", *got, want)
		}
	})

	t.Run("file not found", func(t *testing.T) {
		_, err := ParseConfig("/tmp/nonexistent_janus_test_config.yaml")
		if err == nil {
			t.Error("expected error for missing file, got nil")
		}
	})

	t.Run("invalid yaml", func(t *testing.T) {
		path, cleanup := writeTempFile(t, "test_config_*.yaml", ":\t: bad yaml\n{{{")
		defer cleanup()

		_, err := ParseConfig(path)
		if err == nil {
			t.Error("expected error for invalid YAML, got nil")
		}
	})
}

func TestGenerateFunctionConfig(t *testing.T) {
	t.Run("valid python function", func(t *testing.T) {
		pyPath, pyCleanup := writeTempFile(t, "test_*.py",
			"def greet(name: str, times: int = 3):\n    pass\n")
		defer pyCleanup()

		// ScrollPath shares pyPath's directory so the resulting TargetFile is
		// just the basename — predictable across test runs.
		scrollPath := filepath.Join(filepath.Dir(pyPath), "scroll.yaml")
		g := &ConfigGenerator{AbsPathToFunction: pyPath, ScrollPath: scrollPath, FunctionName: "greet"}
		got, err := g.GenerateFunctionConfig()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		want := types.Function{
			Name:           "greet",
			TargetFile:     filepath.Base(pyPath),
			TargetFunction: "greet",
			Args: []types.Arg{
				{Name: "name", Type: "str", Default: nil},
				{Name: "times", Type: "int", Default: 3},
			},
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("function mismatch\n  got:  %#v\n  want: %#v", got, want)
		}
	})

	t.Run("no file extension", func(t *testing.T) {
		g := &ConfigGenerator{AbsPathToFunction: "script", FunctionName: "run"}
		_, err := g.GenerateFunctionConfig()
		if err == nil {
			t.Fatal("expected error for missing extension, got nil")
		}
		if !strings.Contains(err.Error(), "no file extension found") {
			t.Errorf("expected 'no file extension found' in error, got: %v", err)
		}
	})

	t.Run("unsupported file extension", func(t *testing.T) {
		rbPath, rbCleanup := writeTempFile(t, "test_*.rb", "")
		defer rbCleanup()

		g := &ConfigGenerator{AbsPathToFunction: rbPath, FunctionName: "some_func"}
		_, err := g.GenerateFunctionConfig()
		if err == nil {
			t.Fatal("expected error for unsupported extension, got nil")
		}
		if !strings.Contains(err.Error(), "unsupported file extension") {
			t.Errorf("expected 'unsupported file extension' in error, got: %v", err)
		}
	})

	t.Run("function not found in source", func(t *testing.T) {
		pyPath, pyCleanup := writeTempFile(t, "test_*.py",
			"def other():\n    pass\n")
		defer pyCleanup()

		scrollPath := filepath.Join(filepath.Dir(pyPath), "scroll.yaml")
		g := &ConfigGenerator{AbsPathToFunction: pyPath, ScrollPath: scrollPath, FunctionName: "missing_func"}
		_, err := g.GenerateFunctionConfig()
		if err == nil {
			t.Error("expected error for missing function, got nil")
		}
	})
}
