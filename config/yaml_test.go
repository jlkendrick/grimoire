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
					ScrollPath:    path,
					AbsTargetFile: filepath.Join(filepath.Dir(path), "test/hello_world_func.py"),
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

	// global config flattens its registered scrolls' functions into one slice.
	// Each function's AbsTargetFile must resolve against its OWN scroll's
	// directory, not the global grimoire.yaml's directory — otherwise a
	// scroll registered with the global grimoire from anywhere on disk would
	// fail to resolve its function paths when invoked from an unrelated cwd.
	t.Run("global config resolves AbsTargetFile against each registered scroll's directory", func(t *testing.T) {
		// Build two independent scroll directories, each with its own scroll.yaml
		// pointing at a function file that lives next to it via a relative path.
		scrollDirA := t.TempDir()
		scrollPathA := filepath.Join(scrollDirA, "scroll.yaml")
		if err := os.WriteFile(scrollPathA, []byte("functions:\n- name: a_fn\n  path: src/a.py\n  function: a_fn\n"), 0644); err != nil {
			t.Fatal(err)
		}

		scrollDirB := t.TempDir()
		scrollPathB := filepath.Join(scrollDirB, "scroll.yaml")
		if err := os.WriteFile(scrollPathB, []byte("functions:\n- name: b_fn\n  path: nested/b.go\n  function: BFn\n"), 0644); err != nil {
			t.Fatal(err)
		}

		// Build a global grimoire.yaml that registers both scrolls. The global
		// config sits in its own directory unrelated to either scroll dir.
		globalDir := t.TempDir()
		globalPath := filepath.Join(globalDir, "grimoire.yaml")
		globalContent := "registered_projects:\n- path: " + scrollPathA + "\n- path: " + scrollPathB + "\n"
		if err := os.WriteFile(globalPath, []byte(globalContent), 0644); err != nil {
			t.Fatal(err)
		}

		got, err := ParseConfig(globalPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got.Context != types.ContextTypeGlobal {
			t.Errorf("expected global context, got %q", got.Context)
		}
		if len(got.Functions) != 2 {
			t.Fatalf("expected 2 functions flattened from registered scrolls, got %d", len(got.Functions))
		}

		// Each function's AbsTargetFile must be absolute and rooted in its
		// OWN scroll's directory — not the global grimoire.yaml's directory.
		byName := map[string]types.Function{}
		for _, fn := range got.Functions {
			byName[fn.Name] = fn
		}

		fnA, ok := byName["a_fn"]
		if !ok {
			t.Fatal("a_fn missing from flattened functions")
		}
		wantAbsA := filepath.Join(scrollDirA, "src/a.py")
		if fnA.AbsTargetFile != wantAbsA {
			t.Errorf("a_fn AbsTargetFile = %q, want %q", fnA.AbsTargetFile, wantAbsA)
		}
		if fnA.ScrollPath != scrollPathA {
			t.Errorf("a_fn ScrollPath = %q, want %q", fnA.ScrollPath, scrollPathA)
		}

		fnB, ok := byName["b_fn"]
		if !ok {
			t.Fatal("b_fn missing from flattened functions")
		}
		wantAbsB := filepath.Join(scrollDirB, "nested/b.go")
		if fnB.AbsTargetFile != wantAbsB {
			t.Errorf("b_fn AbsTargetFile = %q, want %q", fnB.AbsTargetFile, wantAbsB)
		}
		if fnB.ScrollPath != scrollPathB {
			t.Errorf("b_fn ScrollPath = %q, want %q", fnB.ScrollPath, scrollPathB)
		}
	})

	// Local config: AbsTargetFile must join against the scroll's parent dir,
	// not the scroll path itself. (Regression test — early refactor mistakenly
	// joined against the .yaml path, producing /.../scroll.yaml/relative/file.py.)
	t.Run("local config resolves AbsTargetFile against scroll's parent directory", func(t *testing.T) {
		dir := t.TempDir()
		scrollPath := filepath.Join(dir, "scroll.yaml")
		if err := os.WriteFile(scrollPath, []byte("functions:\n- name: f\n  path: scripts/f.py\n  function: f\n"), 0644); err != nil {
			t.Fatal(err)
		}

		got, err := ParseConfig(scrollPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got.Functions) != 1 {
			t.Fatalf("expected 1 function, got %d", len(got.Functions))
		}
		want := filepath.Join(dir, "scripts/f.py")
		if got.Functions[0].AbsTargetFile != want {
			t.Errorf("AbsTargetFile = %q, want %q", got.Functions[0].AbsTargetFile, want)
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
