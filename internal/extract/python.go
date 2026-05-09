package extract

import (
	"strings"
	"strconv"

	"github.com/smacker/go-tree-sitter/python"

	sitter "github.com/smacker/go-tree-sitter"
	descriptor "github.com/jlkendrick/grimoire/internal/descriptor"
)

// unquotePythonString strips surrounding matching quotes from a Python string
// literal's source text (e.g. `"world"` -> `world`, `'''hi'''` -> `hi`). It
// only acts on the outer pair; escape sequences inside are left untouched.
// Strings with prefixes (r/b/f/u, possibly combined) keep the prefix.
func unquotePythonString(raw string) string {
	if len(raw) >= 6 {
		if (strings.HasPrefix(raw, `"""`) && strings.HasSuffix(raw, `"""`)) ||
			(strings.HasPrefix(raw, `'''`) && strings.HasSuffix(raw, `'''`)) {
			return raw[3 : len(raw)-3]
		}
	}
	if len(raw) >= 2 {
		first, last := raw[0], raw[len(raw)-1]
		if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
			return raw[1 : len(raw)-1]
		}
	}
	return raw
}

// parsePythonLiteral converts a literal AST node into a Go value plus a
// TypeInfo. Recognized: string, integer, float, true/false, none. For
// anything else (lists, dicts, calls, identifiers) returns the raw source
// text and a nil TypeInfo, leaving type resolution to a later pass.
func parsePythonLiteral(n *sitter.Node, src []byte) (any, *descriptor.TypeInfo) {
	prim := func(name string) *descriptor.TypeInfo {
		return &descriptor.TypeInfo{Kind: descriptor.TypeKindPrimitive, Name: name}
	}
	content := string(n.Content(src))
	switch n.Type() {
	case "string":
		return unquotePythonString(content), prim("str")
	case "integer":
		v, err := strconv.ParseInt(content, 10, 64)
		if err != nil {
			return content, nil
		}
		return v, prim("int")
	case "float":
		v, err := strconv.ParseFloat(content, 64)
		if err != nil {
			return content, nil
		}
		return v, prim("float")
	case "true", "false":
		return n.Type() == "true", prim("bool")
	case "none":
		return nil, nil
	}
	return content, nil
}

var pythonConfig = grammarConfig{
	language:         python.GetLanguage,
	functionNodeType: "function_definition",
	parametersField:  "parameters",
	extractParam:     extractPythonParam,
}

type PythonExtractor struct{}

func (a *PythonExtractor) GenerateDescriptor_ParamsOnly(abs_path_to_function, funcName string) (descriptor.FunctionDescriptor, error) {
	return generateDescriptorBase_ParamsOnly(pythonConfig, abs_path_to_function, funcName)
}

func extractPythonParam(n *sitter.Node, src []byte) []descriptor.ParamDescriptor {
	switch n.Type() {
	case "identifier":
		// def f(x):
		name := string(n.Content(src))
		return []descriptor.ParamDescriptor{
			{
				Name: name,
				RawTypeText: "",
				ResolvedType: nil,
				Default: nil,
				ExtractorNotes: []string{"missing type hint and default value"},
			},
		}

	case "default_parameter":
		// def f(x=1):
		nameNode := n.ChildByFieldName("name")
		valueNode := n.ChildByFieldName("value")
		if nameNode == nil {
			return nil
		}
		var default_value any
		var resolved_type *descriptor.TypeInfo
		if valueNode != nil {
			default_value, resolved_type = parsePythonLiteral(valueNode, src)
		}
		return []descriptor.ParamDescriptor{{
			Name: string(nameNode.Content(src)),
			ResolvedType: resolved_type, // may be nil if not inferred
			Default: default_value,
			ExtractorNotes: []string{"missing type hint"},
		}}

	case "typed_parameter":
		// def f(x: int):
		var name string
		var typ string
		for i := 0; i < int(n.NamedChildCount()); i++ {
			child := n.NamedChild(i)
			if child == nil {
				continue
			}
			if child.Type() == "identifier" {
				name = string(child.Content(src))
			} else {
				typ = string(child.Content(src))
			}
		}
		if name == "" {
			return nil
		}
		return []descriptor.ParamDescriptor{{
			Name: name,
			RawTypeText: typ,
			ResolvedType: &descriptor.TypeInfo{
				Name: typ,
			},
			Default: nil,
			ExtractorNotes: []string{"missing default value"},
		}}

	case "typed_default_parameter":
		// def f(x: int = 1):
		nameNode := n.ChildByFieldName("name")
		typeNode := n.ChildByFieldName("type")
		valueNode := n.ChildByFieldName("value")
		if nameNode == nil {
			return nil
		}

		var typ string
		var resolved_type *descriptor.TypeInfo
		if typeNode != nil {
			typ = string(typeNode.Content(src))
			resolved_type = &descriptor.TypeInfo{Name: typ}
			switch typ {
			case "str", "int", "float", "bool":
				resolved_type.Kind = descriptor.TypeKindPrimitive
			}
		}

		var default_value any
		if valueNode != nil {
			default_value, _ = parsePythonLiteral(valueNode, src)
		}

		return []descriptor.ParamDescriptor{{
			Name: string(nameNode.Content(src)),
			RawTypeText: typ,
			ResolvedType: resolved_type,
			Default: default_value,
			ExtractorNotes: []string{},
		}}

	case "list_splat_pattern": // *args
		if n.NamedChildCount() == 0 {
			return nil
		}
		name := "*" + string(n.NamedChild(0).Content(src))
		return []descriptor.ParamDescriptor{
			{
				Name: name,
				RawTypeText: "",
				ResolvedType: nil,
				Default: nil,
				ExtractorNotes: []string{"missing type hint and default value"},
			},
		}

	case "dictionary_splat_pattern": // **kwargs
		if n.NamedChildCount() == 0 {
			return nil
		}
		name := "**" + string(n.NamedChild(0).Content(src))
		return []descriptor.ParamDescriptor{{
			Name: name,
			RawTypeText: "",
			ResolvedType: nil,
			Default: nil,
			ExtractorNotes: []string{"missing type hint and default value"},
		}}
	}

	// Unsupported param kind (pos-only marker '/', etc.)
	return nil
}
