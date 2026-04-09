package core

import (
	"errors"
	"strings"
	"testing"

	types "github.com/jlkendrick/sigil/types"
)

func TestPythonAdapter_GenerateCommand(t *testing.T) {
	tests := []struct {
		name           string
		targetFile     string
		targetFunction string
		wantTargetDir  string
		wantModule     string
	}{
		{
			name:           "simple_relative_path",
			targetFile:     "sample/hello_world_func.py",
			targetFunction: "hello_world",
			wantTargetDir:  "sample",
			wantModule:     "hello_world_func",
		},
		{
			name:           "nested_path",
			targetFile:     "a/b/c/my_func.py",
			targetFunction: "run",
			wantTargetDir:  "a/b/c",
			wantModule:     "my_func",
		},
		{
			name:           "absolute_path",
			targetFile:     "/home/user/scripts/processor.py",
			targetFunction: "process",
			wantTargetDir:  "/home/user/scripts",
			wantModule:     "processor",
		},
		{
			name:           "flat_no_directory",
			targetFile:     "flat.py",
			targetFunction: "main",
			wantTargetDir:  ".",
			wantModule:     "flat",
		},
		{
			// Documents existing behavior: TrimSuffix removes only ".py",
			// leaving dots in the stem intact. importlib would fail at
			// runtime on such a path, but GenerateCommand itself does not.
			name:           "file_with_multiple_dots",
			targetFile:     "src/my.util.helper.py",
			targetFunction: "compute",
			wantTargetDir:  "src",
			wantModule:     "my.util.helper",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			adapter := &PythonAdapter{}
			fn := types.Function{
				TargetFile:     tc.targetFile,
				TargetFunction: tc.targetFunction,
			}

			binary, flags, err := adapter.GenerateCommand(fn)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if binary != "python" {
				t.Errorf("expected binary %q, got %q", "python", binary)
			}
			if len(flags) != 2 {
				t.Fatalf("expected 2 flags, got %d: %v", len(flags), flags)
			}
			if flags[0] != "-c" {
				t.Errorf("expected flags[0]==-c, got %q", flags[0])
			}

			script := flags[1]

			// Shared structural assertions — present in every generated script.
			for _, want := range []string{
				"importlib.import_module",
				"sys.stdin.read",
				"getattr",
			} {
				if !strings.Contains(script, want) {
					t.Errorf("script missing %q", want)
				}
			}

			// Case-specific: verify the three Sprintf substitutions.
			wantDirLiteral := "os.path.expanduser('" + tc.wantTargetDir + "')"
			if !strings.Contains(script, wantDirLiteral) {
				t.Errorf("script missing target_dir literal %q\nscript:\n%s", wantDirLiteral, script)
			}

			wantModLiteral := "importlib.import_module('" + tc.wantModule + "')"
			if !strings.Contains(script, wantModLiteral) {
				t.Errorf("script missing module literal %q\nscript:\n%s", wantModLiteral, script)
			}

			wantFnLiteral := "getattr(mod, '" + tc.targetFunction + "')"
			if !strings.Contains(script, wantFnLiteral) {
				t.Errorf("script missing function literal %q\nscript:\n%s", wantFnLiteral, script)
			}
		})
	}
}

func TestPythonAdapter_FormatError(t *testing.T) {
	adapter := &PythonAdapter{}

	t.Run("wraps_error_with_prefix", func(t *testing.T) {
		wrapped := adapter.FormatError(errors.New("something went wrong"))
		if wrapped.Error() != "python runtime error: something went wrong" {
			t.Errorf("unexpected error string: %q", wrapped.Error())
		}
	})

	t.Run("preserves_original_message", func(t *testing.T) {
		wrapped := adapter.FormatError(errors.New("exit status 1"))
		if !strings.HasPrefix(wrapped.Error(), "python runtime error:") {
			t.Errorf("missing prefix in: %q", wrapped.Error())
		}
		if !strings.Contains(wrapped.Error(), "exit status 1") {
			t.Errorf("original message not preserved in: %q", wrapped.Error())
		}
	})
}
