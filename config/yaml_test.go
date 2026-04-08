package config

import (
	"os"
	"reflect"
	"strings"
	"testing"

	types "github.com/jlkendrick/sigil/types"

	"github.com/goccy/go-yaml"
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

func TestParseUserConfig(t *testing.T) {
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

		got, err := ParseUserConfig(path)
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
				},
			},
		}

		if !reflect.DeepEqual(*got, want) {
			t.Errorf("config mismatch\n  got:  %#v\n  want: %#v", *got, want)
		}
	})

	t.Run("file not found", func(t *testing.T) {
		_, err := ParseUserConfig("/tmp/nonexistent_janus_test_config.yaml")
		if err == nil {
			t.Error("expected error for missing file, got nil")
		}
	})

	t.Run("invalid yaml", func(t *testing.T) {
		path, cleanup := writeTempFile(t, "test_config_*.yaml", ":\t: bad yaml\n{{{")
		defer cleanup()

		_, err := ParseUserConfig(path)
		if err == nil {
			t.Error("expected error for invalid YAML, got nil")
		}
	})
}

func TestGenerateTypedYAML(t *testing.T) {
	t.Run("args extracted and written back", func(t *testing.T) {
		pyPath, pyCleanup := writeTempFile(t, "test_*.py",
			"def greet(name: str, times: int = 3):\n    pass\n")
		defer pyCleanup()

		cfgPath, cfgCleanup := writeTempFile(t, "test_config_*.yaml", "")
		defer cfgCleanup()

		cfg := &types.Config{
			Functions: []types.Function{
				{
					Name:           "greet",
					TargetFile:     pyPath,
					TargetFunction: "greet",
				},
			},
		}

		g := &ConfigGenerator{ConfigPath: cfgPath, Config: cfg}
		if err := g.GenerateTypedYAML(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		wantArgs := []types.Arg{
			{Name: "name", Type: "str", Default: nil},
			{Name: "times", Type: "int", Default: "3"},
		}

		// Verify in-memory update
		if !reflect.DeepEqual(g.Config.Functions[0].Args, wantArgs) {
			t.Errorf("in-memory args mismatch\n  got:  %#v\n  want: %#v",
				g.Config.Functions[0].Args, wantArgs)
		}

		// Verify round-trip through written YAML
		written, err := os.ReadFile(cfgPath)
		if err != nil {
			t.Fatalf("reading written config: %v", err)
		}
		var roundTripped types.Config
		if err := yaml.Unmarshal(written, &roundTripped); err != nil {
			t.Fatalf("unmarshaling written config: %v", err)
		}
		if !reflect.DeepEqual(roundTripped.Functions[0].Args, wantArgs) {
			t.Errorf("round-trip args mismatch\n  got:  %#v\n  want: %#v",
				roundTripped.Functions[0].Args, wantArgs)
		}
	})

	t.Run("raw script skipped", func(t *testing.T) {
		pyPath, pyCleanup := writeTempFile(t, "test_*.py", "n = 1\nprint(n)\n")
		defer pyCleanup()

		cfgPath, cfgCleanup := writeTempFile(t, "test_config_*.yaml", "")
		defer cfgCleanup()

		cfg := &types.Config{
			Functions: []types.Function{
				{
					Name:           "script",
					TargetFile:     pyPath,
					TargetFunction: "", // raw script — must be skipped
				},
			},
		}

		g := &ConfigGenerator{ConfigPath: cfgPath, Config: cfg}
		if err := g.GenerateTypedYAML(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Args should remain nil — the function was never passed to the analyzer
		if g.Config.Functions[0].Args != nil {
			t.Errorf("expected nil args for raw script, got %#v", g.Config.Functions[0].Args)
		}
	})

	t.Run("unsupported file extension", func(t *testing.T) {
		rbPath, rbCleanup := writeTempFile(t, "test_*.rb", "")
		defer rbCleanup()

		cfgPath, cfgCleanup := writeTempFile(t, "test_config_*.yaml", "")
		defer cfgCleanup()

		cfg := &types.Config{
			Functions: []types.Function{
				{
					Name:           "f",
					TargetFile:     rbPath,
					TargetFunction: "some_func",
				},
			},
		}

		g := &ConfigGenerator{ConfigPath: cfgPath, Config: cfg}
		err := g.GenerateTypedYAML()
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

		cfgPath, cfgCleanup := writeTempFile(t, "test_config_*.yaml", "")
		defer cfgCleanup()

		cfg := &types.Config{
			Functions: []types.Function{
				{
					Name:           "f",
					TargetFile:     pyPath,
					TargetFunction: "missing_func",
				},
			},
		}

		g := &ConfigGenerator{ConfigPath: cfgPath, Config: cfg}
		if err := g.GenerateTypedYAML(); err == nil {
			t.Error("expected error for missing function, got nil")
		}
	})
}
