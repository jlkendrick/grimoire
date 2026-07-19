package extract

import (
	"os"
	"path/filepath"
	"testing"

	descriptor "github.com/jlkendrick/grimoire/internal/descriptor"
)

func extractReturns(t *testing.T, ext LanguageExtractor, filename, src, fn string) *descriptor.TypeInfo {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(src), 0644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	fd, err := ext.GenerateDescriptor(path, fn)
	if err != nil {
		t.Fatalf("GenerateDescriptor: %v", err)
	}
	return fd.Returns
}

// typeShape renders a TypeInfo chain compactly for assertions:
// kind(name)/elem-kind(elem-name)/...
func typeShape(ti *descriptor.TypeInfo) string {
	if ti == nil {
		return "<nil>"
	}
	s := string(ti.Kind)
	if ti.Kind == descriptor.TypeKindPrimitive || ti.Kind == descriptor.TypeKindStruct {
		s += "(" + ti.Name + ")"
	}
	if ti.Element != nil {
		s += "/" + typeShape(ti.Element)
	}
	return s
}

func TestPythonReturnExtraction(t *testing.T) {
	for _, tc := range []struct {
		name string
		src  string
		want string
	}{
		{"primitive", "def f(x: int) -> int:\n    return x\n", "primitive(int)"},
		{"unannotated", "def f(x: int):\n    return x\n", "<nil>"},
		{"none", "def f() -> None:\n    pass\n", "none"},
		{"dict of int", "def f() -> dict[str, int]:\n    return {}\n", "map/primitive(int)"},
		{"nested composition", "def f() -> list[dict[str, float]]:\n    return []\n", "list/map/primitive(float)"},
		{"optional", "def f() -> Optional[str]:\n    return None\n", "optional/primitive(str)"},
		{"union with none", "def f() -> int | None:\n    return None\n", "optional/primitive(int)"},
		{"bare dict has unknown element", "def f() -> dict:\n    return {}\n", "map"},
		// tuple is outside the v1 inventory: it classifies as an opaque
		// struct, which the type algebra treats as Unknown.
		{"tuple is opaque", "def f() -> tuple[int, str]:\n    return 1, \"a\"\n", "struct(tuple[int, str])"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := extractReturns(t, &PythonExtractor{}, "f.py", tc.src, "f")
			if typeShape(got) != tc.want {
				t.Errorf("Returns = %s, want %s", typeShape(got), tc.want)
			}
		})
	}
}

func TestGoReturnExtraction(t *testing.T) {
	for _, tc := range []struct {
		name string
		src  string
		want string
	}{
		{"single type", "package p\n\nfunc F(x int) int { return x }\n", "primitive(int)"},
		{"no result", "package p\n\nfunc F(x int) { }\n", "<nil>"},
		{"t and error yields t", "package p\n\nfunc F() ([]string, error) { return nil, nil }\n", "list/primitive(string)"},
		{"map result", "package p\n\nfunc F() map[string]int { return nil }\n", "map/primitive(int)"},
		{"pointer is optional", "package p\n\nfunc F() *int { return nil }\n", "optional/primitive(int)"},
		{"non-error multi is unknown", "package p\n\nfunc F() (int, string) { return 0, \"\" }\n", "<nil>"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := extractReturns(t, &GoExtractor{}, "f.go", tc.src, "F")
			if typeShape(got) != tc.want {
				t.Errorf("Returns = %s, want %s", typeShape(got), tc.want)
			}
		})
	}
}
