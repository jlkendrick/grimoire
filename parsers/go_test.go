package parsers

import (
	"os"
	"reflect"
	"testing"

	types "github.com/jlkendrick/grimoire/types"
)

func writeTempGoFile(t *testing.T, src string) (path string, cleanup func()) {
	t.Helper()
	f, err := os.CreateTemp("", "test_*.go")
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

func TestGoAnalyzer_ExtractSignature(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		funcName string
		wantArgs []types.Arg
		wantErr  bool
	}{
		{
			name:     "single_param",
			src:      "package p\n\nfunc f(x int) int { return x }\n",
			funcName: "f",
			wantArgs: []types.Arg{{Name: "x", Type: "int"}},
		},
		{
			name:     "multiple_params_same_type",
			src:      "package p\n\nfunc f(x, y int) int { return x + y }\n",
			funcName: "f",
			wantArgs: []types.Arg{
				{Name: "x", Type: "int"},
				{Name: "y", Type: "int"},
			},
		},
		{
			name:     "mixed_types",
			src:      "package p\n\nfunc f(x int, y string, z bool) {}\n",
			funcName: "f",
			wantArgs: []types.Arg{
				{Name: "x", Type: "int"},
				{Name: "y", Type: "string"},
				{Name: "z", Type: "bool"},
			},
		},
		{
			name:     "three_params_same_type",
			src:      "package p\n\nfunc f(a, b, c int) {}\n",
			funcName: "f",
			wantArgs: []types.Arg{
				{Name: "a", Type: "int"},
				{Name: "b", Type: "int"},
				{Name: "c", Type: "int"},
			},
		},
		{
			name:     "variadic_param",
			src:      "package p\n\nfunc f(args ...int) {}\n",
			funcName: "f",
			wantArgs: []types.Arg{{Name: "args", Type: "...int"}},
		},
		{
			name:     "mixed_regular_and_variadic",
			src:      "package p\n\nfunc f(n int, args ...string) {}\n",
			funcName: "f",
			wantArgs: []types.Arg{
				{Name: "n", Type: "int"},
				{Name: "args", Type: "...string"},
			},
		},
		{
			name:     "no_params",
			src:      "package p\n\nfunc f() {}\n",
			funcName: "f",
			wantArgs: []types.Arg{},
		},
		{
			name:     "unnamed_param",
			src:      "package p\n\nfunc f(int) {}\n",
			funcName: "f",
			wantArgs: []types.Arg{{Name: "", Type: "int"}},
		},
		{
			name:     "string_and_float64",
			src:      "package p\n\nfunc f(msg string, ratio float64) {}\n",
			funcName: "f",
			wantArgs: []types.Arg{
				{Name: "msg", Type: "string"},
				{Name: "ratio", Type: "float64"},
			},
		},
		{
			name:     "function_not_found",
			src:      "package p\n\nfunc f() {}\n",
			funcName: "does_not_exist",
			wantErr:  true,
		},
		{
			name:     "correct_function_among_multiple",
			src:      "package p\n\nfunc other(x int) {}\n\nfunc target(a, b string) string { return a + b }\n",
			funcName: "target",
			wantArgs: []types.Arg{
				{Name: "a", Type: "string"},
				{Name: "b", Type: "string"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path, cleanup := writeTempGoFile(t, tc.src)
			defer cleanup()

			analyzer := &GoAnalyzer{}
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
