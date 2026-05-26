package extract

import (
	"strings"

	descriptor "github.com/jlkendrick/grimoire/internal/descriptor"
)

// classifyPythonType maps a Python type-annotation string to a TypeInfo,
// populating Kind and (for collections/optionals) Element. Recognized shapes:
// list[T]/List[T] -> List, dict[K,V]/Dict[K,V] -> Map (Element is the value
// type; JSON keys are strings so the key type is kept only in Name),
// Optional[T] and "T | None" -> Optional, the four primitives -> Primitive,
// and any other identifier (a class) -> Struct. Returns nil for empty input.
func classifyPythonType(raw string) *descriptor.TypeInfo {
	s := strings.TrimSpace(raw)
	if s == "" {
		return nil
	}

	if inner, ok := stripGeneric(s, "Optional"); ok {
		return &descriptor.TypeInfo{Kind: descriptor.TypeKindOptional, Element: classifyPythonType(inner), Name: s}
	}
	if inner, ok := unionWithNone(s); ok {
		return &descriptor.TypeInfo{Kind: descriptor.TypeKindOptional, Element: classifyPythonType(inner), Name: s}
	}
	if inner, ok := stripGeneric(s, "list"); ok {
		return &descriptor.TypeInfo{Kind: descriptor.TypeKindList, Element: classifyPythonType(inner), Name: s}
	}
	if inner, ok := stripGeneric(s, "List"); ok {
		return &descriptor.TypeInfo{Kind: descriptor.TypeKindList, Element: classifyPythonType(inner), Name: s}
	}
	if inner, ok := stripGeneric(s, "dict"); ok {
		return &descriptor.TypeInfo{Kind: descriptor.TypeKindMap, Element: classifyPythonType(mapValueParam(inner)), Name: s}
	}
	if inner, ok := stripGeneric(s, "Dict"); ok {
		return &descriptor.TypeInfo{Kind: descriptor.TypeKindMap, Element: classifyPythonType(mapValueParam(inner)), Name: s}
	}

	switch s {
	case "str", "int", "float", "bool":
		return &descriptor.TypeInfo{Kind: descriptor.TypeKindPrimitive, Name: s}
	}
	return &descriptor.TypeInfo{Kind: descriptor.TypeKindStruct, Name: s}
}

// classifyGoType maps a Go type string to a TypeInfo. Recognized shapes:
// "...T"/"[]T" -> List, "map[K]V" -> Map, "*T" -> Optional, the built-in
// scalar types -> Primitive, and any named type -> Struct.
func classifyGoType(raw string) *descriptor.TypeInfo {
	s := strings.TrimSpace(raw)
	if s == "" {
		return nil
	}

	if rest, ok := strings.CutPrefix(s, "..."); ok {
		return &descriptor.TypeInfo{Kind: descriptor.TypeKindList, Element: classifyGoType(rest), Name: s}
	}
	if rest, ok := strings.CutPrefix(s, "[]"); ok {
		return &descriptor.TypeInfo{Kind: descriptor.TypeKindList, Element: classifyGoType(rest), Name: s}
	}
	if rest, ok := strings.CutPrefix(s, "*"); ok {
		return &descriptor.TypeInfo{Kind: descriptor.TypeKindOptional, Element: classifyGoType(rest), Name: s}
	}
	if rest, ok := strings.CutPrefix(s, "map["); ok {
		// rest is "K]V" (K may itself contain brackets). Find the bracket that
		// closes the key at depth 0; everything after it is the value type.
		depth := 0
		for i := 0; i < len(rest); i++ {
			switch rest[i] {
			case '[':
				depth++
			case ']':
				if depth == 0 {
					return &descriptor.TypeInfo{Kind: descriptor.TypeKindMap, Element: classifyGoType(rest[i+1:]), Name: s}
				}
				depth--
			}
		}
	}

	switch s {
	case "string", "bool", "rune", "byte",
		"int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
		"float32", "float64", "complex64", "complex128":
		return &descriptor.TypeInfo{Kind: descriptor.TypeKindPrimitive, Name: s}
	}
	return &descriptor.TypeInfo{Kind: descriptor.TypeKindStruct, Name: s}
}

// stripGeneric returns the inner text of `name[...]` (e.g. stripGeneric("list[int]",
// "list") -> "int", true). The closing bracket must be the final character.
func stripGeneric(s, name string) (string, bool) {
	prefix := name + "["
	if strings.HasPrefix(s, prefix) && strings.HasSuffix(s, "]") {
		return s[len(prefix) : len(s)-1], true
	}
	return "", false
}

// mapValueParam returns the value type from a dict's "K, V" parameter list,
// splitting on the first top-level comma (bracket-depth aware). If there's no
// top-level comma it returns the whole string.
func mapValueParam(inner string) string {
	depth := 0
	for i := 0; i < len(inner); i++ {
		switch inner[i] {
		case '[':
			depth++
		case ']':
			depth--
		case ',':
			if depth == 0 {
				return strings.TrimSpace(inner[i+1:])
			}
		}
	}
	return strings.TrimSpace(inner)
}

// unionWithNone detects a two-arm "T | None" optional and returns T. Anything
// more complex (multiple non-None arms) is left for the caller to treat as a
// struct/unknown.
func unionWithNone(s string) (string, bool) {
	if !strings.Contains(s, "|") {
		return "", false
	}
	depth := 0
	var arms []string
	start := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '[':
			depth++
		case ']':
			depth--
		case '|':
			if depth == 0 {
				arms = append(arms, strings.TrimSpace(s[start:i]))
				start = i + 1
			}
		}
	}
	arms = append(arms, strings.TrimSpace(s[start:]))
	if len(arms) != 2 {
		return "", false
	}
	switch {
	case arms[0] == "None":
		return arms[1], true
	case arms[1] == "None":
		return arms[0], true
	}
	return "", false
}
