package core

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"

	types "github.com/jlkendrick/grimoire/types"
)

// writeTempPyFile creates a temporary Python file with the given source and
// returns its path along with a cleanup function.
func writeTempPyFile(t *testing.T, src string) (path string, cleanup func()) {
	t.Helper()
	f, err := os.CreateTemp("", "test_*.py")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	if _, err := f.WriteString(src); err != nil {
		f.Close()
		os.Remove(f.Name())
		t.Fatalf("WriteString: %v", err)
	}
	f.Close()
	return f.Name(), func() { os.Remove(f.Name()) }
}

// requirePython skips the test if the "python" binary is not on PATH.
// The adapter hardcodes "python", so integration tests must skip rather
// than fail on systems where only "python3" is available.
func requirePython(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("python"); err != nil {
		t.Skip("python binary not found on PATH; skipping integration test")
	}
}

func TestAssignAdapter(t *testing.T) {
	tests := []struct {
		name            string
		targetFile      string
		wantErr         bool
		wantErrContains string
	}{
		{
			name:       "py_extension",
			targetFile: "script.py",
			wantErr:    false,
		},
		{
			name:            "no_extension",
			targetFile:      "scriptnoext",
			wantErr:         true,
			wantErrContains: "no file extension found",
		},
		{
			name:            "unsupported_rb",
			targetFile:      "script.rb",
			wantErr:         true,
			wantErrContains: "unsupported file extension",
		},
		{
			name:            "unsupported_js",
			targetFile:      "script.js",
			wantErr:         true,
			wantErrContains: "unsupported file extension",
		},
		{
			// strings.Split on "my.util.helper.py" yields last element "py",
			// so this correctly maps to PythonAdapter.
			name:       "multiple_dots_py",
			targetFile: "my.util.helper.py",
			wantErr:    false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fn := types.Function{TargetFile: tc.targetFile}
			adapter, err := assignAdapter(fn)

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tc.wantErrContains != "" && !strings.Contains(err.Error(), tc.wantErrContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tc.wantErrContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if adapter == nil {
				t.Error("expected non-nil adapter, got nil")
			}
		})
	}
}

func TestExecuteFunction(t *testing.T) {
	t.Run("returns_string", func(t *testing.T) {
		requirePython(t)
		path, cleanup := writeTempPyFile(t, "def greet(name):\n    return 'hello ' + name\n")
		defer cleanup()

		out, err := ExecuteFunction(
			types.Function{TargetFile: path, TargetFunction: "greet"},
			map[string]interface{}{"name": "world"},
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := strings.TrimSpace(string(out)); got != "hello world" {
			t.Errorf("expected %q, got %q", "hello world", got)
		}
	})

	t.Run("returns_none_produces_empty_output", func(t *testing.T) {
		requirePython(t)
		path, cleanup := writeTempPyFile(t, "def noop():\n    pass\n")
		defer cleanup()

		out, err := ExecuteFunction(
			types.Function{TargetFile: path, TargetFunction: "noop"},
			map[string]interface{}{},
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := strings.TrimSpace(string(out)); got != "" {
			t.Errorf("expected empty output, got %q", got)
		}
	})

	t.Run("returns_dict_produces_json", func(t *testing.T) {
		requirePython(t)
		path, cleanup := writeTempPyFile(t,
			"def make_dict(k, v):\n    return {'k': k, 'v': v}\n")
		defer cleanup()

		out, err := ExecuteFunction(
			types.Function{TargetFile: path, TargetFunction: "make_dict"},
			map[string]interface{}{"k": "x", "v": 99},
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var got map[string]interface{}
		if err := json.Unmarshal(out, &got); err != nil {
			t.Fatalf("output is not valid JSON: %v\noutput: %s", err, out)
		}
		if got["k"] != "x" {
			t.Errorf("expected k=%q, got %v", "x", got["k"])
		}
		if got["v"] != float64(99) {
			t.Errorf("expected v=float64(99), got %v (%T)", got["v"], got["v"])
		}
	})

	t.Run("returns_list_produces_json", func(t *testing.T) {
		requirePython(t)
		path, cleanup := writeTempPyFile(t,
			"def make_list(n):\n    return list(range(int(n)))\n")
		defer cleanup()

		out, err := ExecuteFunction(
			types.Function{TargetFile: path, TargetFunction: "make_list"},
			map[string]interface{}{"n": 3},
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var got []interface{}
		if err := json.Unmarshal(out, &got); err != nil {
			t.Fatalf("output is not valid JSON: %v\noutput: %s", err, out)
		}
		if len(got) != 3 {
			t.Fatalf("expected 3 elements, got %d", len(got))
		}
		if got[0] != float64(0) {
			t.Errorf("expected got[0]==float64(0), got %v (%T)", got[0], got[0])
		}
	})

	t.Run("int_args_passed_correctly", func(t *testing.T) {
		requirePython(t)
		path, cleanup := writeTempPyFile(t, "def add(a, b):\n    return a + b\n")
		defer cleanup()

		out, err := ExecuteFunction(
			types.Function{TargetFile: path, TargetFunction: "add"},
			map[string]interface{}{"a": 3, "b": 4},
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := strings.TrimSpace(string(out)); got != "7" {
			t.Errorf("expected %q, got %q", "7", got)
		}
	})

	t.Run("string_args_passed_correctly", func(t *testing.T) {
		requirePython(t)
		path, cleanup := writeTempPyFile(t, "def repeat(s, times):\n    return s * times\n")
		defer cleanup()

		out, err := ExecuteFunction(
			types.Function{TargetFile: path, TargetFunction: "repeat"},
			map[string]interface{}{"s": "ab", "times": 3},
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := strings.TrimSpace(string(out)); got != "ababab" {
			t.Errorf("expected %q, got %q", "ababab", got)
		}
	})

	t.Run("function_raises_exception_wraps_error", func(t *testing.T) {
		requirePython(t)
		path, cleanup := writeTempPyFile(t, "def boom():\n    raise ValueError('intentional error')\n")
		defer cleanup()

		_, err := ExecuteFunction(
			types.Function{TargetFile: path, TargetFunction: "boom"},
			map[string]interface{}{},
		)
		if err == nil {
			t.Fatal("expected error from raising function, got nil")
		}
		if !strings.Contains(err.Error(), "python runtime error:") {
			t.Errorf("error should contain 'python runtime error:', got: %q", err.Error())
		}
	})

	// The next two cases never reach the subprocess — assignAdapter fails
	// before exec.Command is called, so requirePython is not needed.

	t.Run("no_extension_returns_error", func(t *testing.T) {
		_, err := ExecuteFunction(
			types.Function{TargetFile: "noextension", TargetFunction: "f"},
			map[string]interface{}{},
		)
		if err == nil {
			t.Fatal("expected error for missing file extension, got nil")
		}
		if !strings.Contains(err.Error(), "no file extension found") {
			t.Errorf("error should contain 'no file extension found', got: %q", err.Error())
		}
	})

	t.Run("unsupported_extension_returns_error", func(t *testing.T) {
		_, err := ExecuteFunction(
			types.Function{TargetFile: "script.rb", TargetFunction: "f"},
			map[string]interface{}{},
		)
		if err == nil {
			t.Fatal("expected error for unsupported extension, got nil")
		}
		if !strings.Contains(err.Error(), "unsupported file extension") {
			t.Errorf("error should contain 'unsupported file extension', got: %q", err.Error())
		}
	})
}
