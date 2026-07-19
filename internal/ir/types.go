package ir

import (
	"fmt"
	"strings"

	descriptor "github.com/jlkendrick/grimoire/internal/descriptor"
	expr "github.com/jlkendrick/grimoire/internal/expr"
)

// This file is the type algebra behind load-time wiring validation:
// walking a reference path through a type, deciding assignability, and
// inferring an expression's type. One principle governs all three —
// gradual typing: a nil *TypeInfo means Unknown, Unknown absorbs every
// operation and is compatible with everything, so annotations only ever
// ADD checking. Unannotated scrolls can never be rejected.
//
// Struct types are opaque in v1 (no field expansion yet) and behave
// exactly like Unknown.

// opaque reports whether t is outside the checkable v1 inventory.
func opaque(t *TypeInfo) bool {
	return t == nil || t.Kind == descriptor.TypeKindUnknown || t.Kind == descriptor.TypeKindStruct
}

// canonicalPrimitive collapses per-language primitive names into the
// canonical set {str, int, float, bool} so cross-language wiring
// compares correctly (Python str ← Go string).
func canonicalPrimitive(name string) string {
	switch name {
	case "str", "string":
		return "str"
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "byte", "rune":
		return "int"
	case "float", "float32", "float64":
		return "float"
	case "bool":
		return "bool"
	}
	return name
}

func describe(t *TypeInfo) string {
	if t == nil {
		return "unknown"
	}
	switch t.Kind {
	case descriptor.TypeKindPrimitive:
		return canonicalPrimitive(t.Name)
	case descriptor.TypeKindList:
		return "list[" + describe(t.Element) + "]"
	case descriptor.TypeKindMap:
		return "dict[str, " + describe(t.Element) + "]"
	case descriptor.TypeKindOptional:
		return "optional[" + describe(t.Element) + "]"
	case descriptor.TypeKindNone:
		return "none"
	case descriptor.TypeKindStruct:
		if t.Name != "" {
			return t.Name
		}
		return "struct"
	}
	return "unknown"
}

// WalkRef follows a reference path's accessors — the part after the root
// binding name, e.g. ".key[0]" of "res.key[0]" — through a type. It
// mirrors expr.ResolveReference's value walk rule-for-rule and the two
// must stay in agreement: `.key` reads a map's element, `[i]` a list's,
// optionals are walked through (the value may be null at runtime, which
// is possibly-wrong, not certainly-wrong), Unknown and opaque types
// absorb any accessor, and an accessor on a primitive or none is a
// definite wiring error.
func WalkRef(t *TypeInfo, accessors string) (*TypeInfo, error) {
	cur := t
	rest := accessors
	for rest != "" {
		// Walk through optional wrappers before applying an accessor.
		for cur != nil && cur.Kind == descriptor.TypeKindOptional {
			cur = cur.Element
		}

		switch rest[0] {
		case '.':
			end := len(rest)
			for i := 1; i < len(rest); i++ {
				if rest[i] == '.' || rest[i] == '[' {
					end = i
					break
				}
			}
			key := rest[1:end]
			rest = rest[end:]

			switch {
			case opaque(cur):
				cur = nil
			case cur.Kind == descriptor.TypeKindMap:
				cur = cur.Element
			default:
				return nil, fmt.Errorf("cannot access field %q on %s", key, describe(cur))
			}

		case '[':
			end := strings.IndexByte(rest, ']')
			if end < 0 {
				return nil, fmt.Errorf("malformed accessor %q", rest)
			}
			idx := rest[1:end]
			rest = rest[end+1:]

			switch {
			case opaque(cur):
				cur = nil
			case cur.Kind == descriptor.TypeKindList:
				cur = cur.Element
			default:
				return nil, fmt.Errorf("cannot index [%s] into %s", idx, describe(cur))
			}

		default:
			return nil, fmt.Errorf("malformed accessor %q", rest)
		}
	}
	return cur, nil
}

// Compatible reports whether a value of type actual can be wired into a
// slot expecting expected. nil means yes. The governing rule: errors
// only for certainly-wrong wiring — Unknown and opaque types pass, an
// optional actual passes into a required slot (possibly null is only
// possibly wrong), but kind mismatches, primitive mismatches, and the
// lossy int ← float are definite errors. float ← int widens silently.
func Compatible(expected, actual *TypeInfo) error {
	if opaque(expected) || opaque(actual) {
		return nil
	}

	// Lenient deref: optional[U] wired into T checks U against T.
	for actual.Kind == descriptor.TypeKindOptional {
		if actual.Element == nil {
			return nil
		}
		actual = actual.Element
	}

	if expected.Kind == descriptor.TypeKindOptional {
		if actual.Kind == descriptor.TypeKindNone {
			return nil
		}
		if expected.Element == nil {
			return nil
		}
		return Compatible(expected.Element, actual)
	}

	if actual.Kind == descriptor.TypeKindNone {
		return fmt.Errorf("cannot wire none into %s", describe(expected))
	}

	switch expected.Kind {
	case descriptor.TypeKindPrimitive:
		if actual.Kind != descriptor.TypeKindPrimitive {
			return fmt.Errorf("cannot wire %s into %s", describe(actual), describe(expected))
		}
		e, a := canonicalPrimitive(expected.Name), canonicalPrimitive(actual.Name)
		if e == a {
			return nil
		}
		if e == "float" && a == "int" {
			return nil // widening
		}
		return fmt.Errorf("cannot wire %s into %s", a, e)

	case descriptor.TypeKindList:
		if actual.Kind != descriptor.TypeKindList {
			return fmt.Errorf("cannot wire %s into %s", describe(actual), describe(expected))
		}
		return Compatible(expected.Element, actual.Element)

	case descriptor.TypeKindMap:
		if actual.Kind != descriptor.TypeKindMap {
			return fmt.Errorf("cannot wire %s into %s", describe(actual), describe(expected))
		}
		return Compatible(expected.Element, actual.Element)

	case descriptor.TypeKindNone:
		return fmt.Errorf("cannot wire %s into none", describe(actual))
	}
	return nil
}

// InferExpr types an expression against a static environment of binding
// name → type. The grammar is tiny, so inference is total: literals type
// themselves, logic yields bool (and requires bool-or-Unknown operands),
// comparisons yield bool (ordering requires numeric-or-Unknown), and
// references walk the environment. A nil result is Unknown.
func InferExpr(e expr.Expr, env map[string]*TypeInfo) (*TypeInfo, error) {
	prim := func(name string) *TypeInfo {
		return &TypeInfo{Kind: descriptor.TypeKindPrimitive, Name: name}
	}

	switch n := e.(type) {
	case expr.LiteralExpr:
		switch n.Value.(type) {
		case bool:
			return prim("bool"), nil
		case float64:
			return prim("float"), nil
		case string:
			return prim("str"), nil
		case nil:
			return &TypeInfo{Kind: descriptor.TypeKindNone}, nil
		}
		return nil, nil

	case expr.ReferenceExpr:
		root := expr.ReferenceRoot(n.Path)
		return WalkRef(env[root], n.Path[len(root):])

	case expr.NotExpr:
		t, err := InferExpr(n.Operand, env)
		if err != nil {
			return nil, err
		}
		if err := requireBool(t, "!"); err != nil {
			return nil, err
		}
		return prim("bool"), nil

	case expr.BinaryExpr:
		left, err := InferExpr(n.Left, env)
		if err != nil {
			return nil, err
		}
		right, err := InferExpr(n.Right, env)
		if err != nil {
			return nil, err
		}
		switch n.Op {
		case "&&", "||":
			if err := requireBool(left, n.Op); err != nil {
				return nil, err
			}
			if err := requireBool(right, n.Op); err != nil {
				return nil, err
			}
		case "<", "<=", ">", ">=":
			if err := requireNumeric(left, n.Op); err != nil {
				return nil, err
			}
			if err := requireNumeric(right, n.Op); err != nil {
				return nil, err
			}
		}
		return prim("bool"), nil
	}
	return nil, nil
}

func requireBool(t *TypeInfo, op string) error {
	if opaque(t) {
		return nil
	}
	if t.Kind == descriptor.TypeKindPrimitive && canonicalPrimitive(t.Name) == "bool" {
		return nil
	}
	return fmt.Errorf("operator %s requires bool operands, got %s", op, describe(t))
}

func requireNumeric(t *TypeInfo, op string) error {
	if opaque(t) {
		return nil
	}
	if t.Kind == descriptor.TypeKindPrimitive {
		if c := canonicalPrimitive(t.Name); c == "int" || c == "float" {
			return nil
		}
	}
	return fmt.Errorf("operator %s requires numeric operands, got %s", op, describe(t))
}
