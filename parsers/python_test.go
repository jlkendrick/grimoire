package parsers

import (
	"os"
	"reflect"
	"testing"

	types "github.com/jlkendrick/grimoire/types"
)

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

func TestPythonAnalyzer_ExtractSignature(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		funcName string
		wantArgs []types.Arg
		wantErr  bool
	}{
		{
			name:     "typed_default_parameter",
			src:      "def hello_world(n: int = 1):\n    pass\n",
			funcName: "hello_world",
			wantArgs: []types.Arg{
				{Name: "n", Type: "int", Default: "1"},
			},
		},
		{
			name:     "identifier_only",
			src:      "def f(x):\n    pass\n",
			funcName: "f",
			wantArgs: []types.Arg{
				{Name: "x", Type: "", Default: nil},
			},
		},
		{
			name:     "default_parameter",
			src:      "def f(x=1):\n    pass\n",
			funcName: "f",
			wantArgs: []types.Arg{
				{Name: "x", Type: "", Default: "1"},
			},
		},
		{
			name:     "typed_parameter",
			src:      "def f(x: str):\n    pass\n",
			funcName: "f",
			wantArgs: []types.Arg{
				{Name: "x", Type: "str", Default: nil},
			},
		},
		{
			name:     "mixed_params",
			src:      "def f(a, b: str, c=1, d: int = 2):\n    pass\n",
			funcName: "f",
			wantArgs: []types.Arg{
				{Name: "a", Type: "", Default: nil},
				{Name: "b", Type: "str", Default: nil},
				{Name: "c", Type: "", Default: "1"},
				{Name: "d", Type: "int", Default: "2"},
			},
		},
		{
			name:     "list_splat_args",
			src:      "def f(*args):\n    pass\n",
			funcName: "f",
			wantArgs: []types.Arg{
				{Name: "*args", Type: "", Default: nil},
			},
		},
		{
			name:     "dictionary_splat_kwargs",
			src:      "def f(**kwargs):\n    pass\n",
			funcName: "f",
			wantArgs: []types.Arg{
				{Name: "**kwargs", Type: "", Default: nil},
			},
		},
		{
			name:     "args_and_kwargs",
			src:      "def f(*args, **kwargs):\n    pass\n",
			funcName: "f",
			wantArgs: []types.Arg{
				{Name: "*args", Type: "", Default: nil},
				{Name: "**kwargs", Type: "", Default: nil},
			},
		},
		{
			name:     "no_params",
			src:      "def f():\n    pass\n",
			funcName: "f",
			wantArgs: []types.Arg{},
		},
		{
			name:     "function_not_found",
			src:      "def f():\n    pass\n",
			funcName: "does_not_exist",
			wantErr:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path, cleanup := writeTempPyFile(t, tc.src)
			defer cleanup()

			analyzer := &PythonAnalyzer{}
			got, err := analyzer.ExtractSignature(path, tc.funcName)

			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tc.wantArgs) {
				t.Errorf("args mismatch\n  got:  %#v\n  want: %#v", got, tc.wantArgs)
			}
		})
	}
}
