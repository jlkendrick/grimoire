package runtimes

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	types "github.com/jlkendrick/grimoire/types"
)

// requireGo skips the test if the "go" binary is not on PATH.
func requireGo(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go binary not found on PATH; skipping integration test")
	}
}

// makeTestGoModule creates a temp dir with a minimal Go module and returns
// the spell path (tempDir/scroll.yaml) and the absolute path to the source
// file. The caller wires both into Function.ScrollPath / Function.AbsTargetFile.
func makeTestGoModule(t *testing.T, moduleName, filename, src string) (scrollPath, absTargetFile string) {
	t.Helper()
	dir := t.TempDir()

	goModContent := "module " + moduleName + "\n\ngo 1.23\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatalf("WriteFile go.mod: %v", err)
	}
	absTargetFile = filepath.Join(dir, filename)
	if err := os.WriteFile(absTargetFile, []byte(src), 0644); err != nil {
		t.Fatalf("WriteFile source: %v", err)
	}
	return filepath.Join(dir, "scroll.yaml"), absTargetFile
}

// -------------------------------------------------------------------------
// Unit tests
// -------------------------------------------------------------------------

func TestGetModuleName(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		wantModule string
		wantErr    bool
	}{
		{
			name:       "standard_module",
			content:    "module github.com/user/repo\n\ngo 1.21\n",
			wantModule: "github.com/user/repo",
		},
		{
			name:       "simple_module_name",
			content:    "module mymod\n\ngo 1.21\n",
			wantModule: "mymod",
		},
		{
			name:       "module_with_leading_comment",
			content:    "// generated file\nmodule example.com/proj\n\ngo 1.21\n",
			wantModule: "example.com/proj",
		},
		{
			name:    "no_module_declaration",
			content: "go 1.21\n",
			wantErr: true,
		},
		{
			name:    "empty_file",
			content: "",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f, err := os.CreateTemp("", "go.mod.*")
			if err != nil {
				t.Fatalf("CreateTemp: %v", err)
			}
			defer os.Remove(f.Name())
			if _, err := f.WriteString(tc.content); err != nil {
				f.Close()
				t.Fatalf("WriteString: %v", err)
			}
			f.Close()

			got, err := getModuleName(f.Name())
			if tc.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.wantModule {
				t.Errorf("got %q, want %q", got, tc.wantModule)
			}
		})
	}
}

func TestUppercaseFirst(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "Hello"},
		{"", ""},
		{"Hello", "Hello"},
		{"x", "X"},
		{"ABC", "ABC"},
		{"_private", "_private"},
	}

	for _, tc := range tests {
		if got := uppercaseFirst(tc.input); got != tc.want {
			t.Errorf("uppercaseFirst(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestGenerateWrapper(t *testing.T) {
	t.Run("with_two_int_args", func(t *testing.T) {
		data := WrapperData{
			UserModule: "mymod",
			FuncName:   "Add",
			Args: []ArgDef{
				{Name: "A", Type: "int", Key: "a"},
				{Name: "B", Type: "int", Key: "b"},
			},
		}
		f, err := os.CreateTemp("", "wrapper_*.go")
		if err != nil {
			t.Fatalf("CreateTemp: %v", err)
		}
		path := f.Name()
		f.Close()
		defer os.Remove(path)

		if err := generateWrapper(path, data); err != nil {
			t.Fatalf("generateWrapper: %v", err)
		}

		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile: %v", err)
		}
		src := string(content)

		for _, want := range []string{
			"package main",
			`userpkg "mymod"`,
			"userpkg.Add",
			"A int",
			"B int",
			`json:"a"`,
			`json:"b"`,
			"reflect.ValueOf(args.A)",
			"reflect.ValueOf(args.B)",
		} {
			if !strings.Contains(src, want) {
				t.Errorf("generated wrapper missing %q\nwrapper:\n%s", want, src)
			}
		}
	})

	t.Run("no_args", func(t *testing.T) {
		data := WrapperData{
			UserModule: "mymod/subpkg",
			FuncName:   "Noop",
			Args:       []ArgDef{},
		}
		f, err := os.CreateTemp("", "wrapper_*.go")
		if err != nil {
			t.Fatalf("CreateTemp: %v", err)
		}
		path := f.Name()
		f.Close()
		defer os.Remove(path)

		if err := generateWrapper(path, data); err != nil {
			t.Fatalf("generateWrapper: %v", err)
		}

		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile: %v", err)
		}
		src := string(content)

		for _, want := range []string{
			"package main",
			`userpkg "mymod/subpkg"`,
			"userpkg.Noop",
		} {
			if !strings.Contains(src, want) {
				t.Errorf("generated wrapper missing %q\nwrapper:\n%s", want, src)
			}
		}
	})
}

func TestGoAdapter_FormatError(t *testing.T) {
	adapter := &GoAdapter{}

	t.Run("wraps_error_with_prefix", func(t *testing.T) {
		wrapped := adapter.FormatError(errors.New("something failed"))
		if !strings.HasPrefix(wrapped.Error(), "error executing go function:") {
			t.Errorf("unexpected error string: %q", wrapped.Error())
		}
	})

	t.Run("preserves_original_message", func(t *testing.T) {
		wrapped := adapter.FormatError(errors.New("exit status 1"))
		if !strings.Contains(wrapped.Error(), "exit status 1") {
			t.Errorf("original message not preserved in: %q", wrapped.Error())
		}
	})
}

func TestGoAdapter_PrepareCommand(t *testing.T) {
	adapter := &GoAdapter{}
	ctx := &ExecutionContext{
		StateMap: map[string]any{
			"args": map[string]interface{}{
				"a": 1,
				"b": 2,
			},
		},
	}

	if err := adapter.PrepareCommand(ctx); err != nil {
		t.Fatalf("PrepareCommand: %v", err)
	}

	flags, ok := ctx.StateMap["flags"].([]string)
	if !ok {
		t.Fatal("flags not set in StateMap")
	}
	if len(flags) != 0 {
		t.Errorf("expected empty flags, got %v", flags)
	}

	jsonArgs, ok := ctx.StateMap["json_args"].([]byte)
	if !ok {
		t.Fatal("json_args not set in StateMap")
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal(jsonArgs, &decoded); err != nil {
		t.Fatalf("json_args not valid JSON: %v", err)
	}
	if decoded["a"] != float64(1) {
		t.Errorf("expected a=1, got %v", decoded["a"])
	}
	if decoded["b"] != float64(2) {
		t.Errorf("expected b=2, got %v", decoded["b"])
	}
}

func TestAssignAdapter_GoExtension(t *testing.T) {
	fn := types.Function{TargetFile: "main.go"}
	adapter, err := assignAdapter(fn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := adapter.(*GoAdapter); !ok {
		t.Errorf("expected *GoAdapter, got %T", adapter)
	}
}

// -------------------------------------------------------------------------
// Integration tests — require `go` on PATH and write to $GRIMOIRE_HOME/envs/
// -------------------------------------------------------------------------

func TestGoRun(t *testing.T) {
	requireGo(t)

	t.Run("returns_int", func(t *testing.T) {
		scrollPath, absTargetFile := makeTestGoModule(t, "testmod_add", "math.go",
			"package testmod_add\n\nfunc Add(a, b int) int { return a + b }\n")

		out, err := Run(
			types.Function{
				ScrollPath:     scrollPath,
				TargetFile:     "math.go",
				AbsTargetFile:  absTargetFile,
				TargetFunction: "Add",
				Args: []types.Arg{
					{Name: "a", Type: "int"},
					{Name: "b", Type: "int"},
				},
			},
			map[string]interface{}{"a": 3, "b": 4},
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := strings.TrimSpace(string(out.Output)); got != "7" {
			t.Errorf("expected %q, got %q", "7", got)
		}
	})

	t.Run("returns_string", func(t *testing.T) {
		scrollPath, absTargetFile := makeTestGoModule(t, "testmod_greet", "greet.go",
			"package testmod_greet\n\nfunc Greet(name string) string { return \"hello \" + name }\n")

		out, err := Run(
			types.Function{
				ScrollPath:     scrollPath,
				TargetFile:     "greet.go",
				AbsTargetFile:  absTargetFile,
				TargetFunction: "Greet",
				Args:           []types.Arg{{Name: "name", Type: "string"}},
			},
			map[string]interface{}{"name": "world"},
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := strings.TrimSpace(string(out.Output)); got != "hello world" {
			t.Errorf("expected %q, got %q", "hello world", got)
		}
	})

	t.Run("no_args_no_return", func(t *testing.T) {
		scrollPath, absTargetFile := makeTestGoModule(t, "testmod_noop", "noop.go",
			"package testmod_noop\n\nfunc Noop() {}\n")

		out, err := Run(
			types.Function{
				ScrollPath:     scrollPath,
				TargetFile:     "noop.go",
				AbsTargetFile:  absTargetFile,
				TargetFunction: "Noop",
				Args:           []types.Arg{},
			},
			map[string]interface{}{},
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := strings.TrimSpace(string(out.Output)); got != "" {
			t.Errorf("expected empty output, got %q", got)
		}
	})

	t.Run("bool_arg", func(t *testing.T) {
		scrollPath, absTargetFile := makeTestGoModule(t, "testmod_bool", "booltest.go",
			"package testmod_bool\n\nfunc Negate(b bool) bool { return !b }\n")

		out, err := Run(
			types.Function{
				ScrollPath:     scrollPath,
				TargetFile:     "booltest.go",
				AbsTargetFile:  absTargetFile,
				TargetFunction: "Negate",
				Args:           []types.Arg{{Name: "b", Type: "bool"}},
			},
			map[string]interface{}{"b": true},
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// bools are not strings so they come out JSON-encoded
		if got := strings.TrimSpace(string(out.Output)); got != "false" {
			t.Errorf("expected %q, got %q", "false", got)
		}
	})

	t.Run("stderr_does_not_contaminate_stdout", func(t *testing.T) {
		src := "package testmod_stderr\n\nimport (\n\t\"fmt\"\n\t\"os\"\n)\n\nfunc LogAndReturn() string {\n\tfmt.Fprintln(os.Stderr, \"log line\")\n\treturn \"value\"\n}\n"
		scrollPath, absTargetFile := makeTestGoModule(t, "testmod_stderr", "logging.go", src)

		out, err := Run(
			types.Function{
				ScrollPath:     scrollPath,
				TargetFile:     "logging.go",
				AbsTargetFile:  absTargetFile,
				TargetFunction: "LogAndReturn",
				Args:           []types.Arg{},
			},
			map[string]interface{}{},
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if strings.Contains(string(out.Output), "log line") {
			t.Errorf("stderr leaked into stdout output: %q", out.Output)
		}
		if got := strings.TrimSpace(string(out.Output)); got != "value" {
			t.Errorf("expected %q, got %q", "value", got)
		}
	})

	t.Run("reuses_cached_binary_on_second_call", func(t *testing.T) {
		scrollPath, absTargetFile := makeTestGoModule(t, "testmod_cache", "math.go",
			"package testmod_cache\n\nfunc Double(n int) int { return n * 2 }\n")

		fn := types.Function{
			ScrollPath:     scrollPath,
			TargetFile:     "math.go",
			AbsTargetFile:  absTargetFile,
			TargetFunction: "Double",
			Args:           []types.Arg{{Name: "n", Type: "int"}},
		}
		args := map[string]interface{}{"n": 5}

		// First call — compiles the binary.
		out1, err := Run(fn, args)
		if err != nil {
			t.Fatalf("first call unexpected error: %v", err)
		}

		// Second call — should reuse the cached binary without recompiling.
		out2, err := Run(fn, args)
		if err != nil {
			t.Fatalf("second call unexpected error: %v", err)
		}

		if string(out1.Output) != string(out2.Output) {
			t.Errorf("outputs differ:\n  first:  %q\n  second: %q", out1, out2)
		}
		if got := strings.TrimSpace(string(out2.Output)); got != "10" {
			t.Errorf("expected %q, got %q", "10", got)
		}
	})
}
