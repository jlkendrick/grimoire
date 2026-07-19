package extract

import (
	"github.com/smacker/go-tree-sitter/golang"

	sitter "github.com/smacker/go-tree-sitter"
	descriptor "github.com/jlkendrick/grimoire/internal/descriptor"
)

var goConfig = grammarConfig{
	language:         golang.GetLanguage,
	functionNodeType: "function_declaration",
	parametersField:  "parameters",
	extractParam:     extractGoParam,
	extractReturn:    extractGoReturn,
}

type GoExtractor struct{}

func (a *GoExtractor) GenerateDescriptor(abs_path_to_function, funcName string) (descriptor.FunctionDescriptor, error) {
	return generateDescriptorBase(goConfig, abs_path_to_function, funcName)
}

// extractGoReturn reads a function_declaration's result. A single type
// yields that type; the idiomatic (T, error) pair yields T (the wrapper
// surfaces the error separately, so T is what flows between steps); any
// other multi-value result is outside the v1 inventory and yields nil
// (Unknown). No result clause yields nil.
func extractGoReturn(fnNode *sitter.Node, src []byte) *descriptor.TypeInfo {
	result := fnNode.ChildByFieldName("result")
	if result == nil {
		return nil
	}
	if result.Type() != "parameter_list" {
		// Bare type: func f() int
		return classifyGoType(string(result.Content(src)))
	}

	var types []string
	for i := 0; i < int(result.NamedChildCount()); i++ {
		child := result.NamedChild(i)
		if child.Type() != "parameter_declaration" {
			continue
		}
		if typeNode := child.ChildByFieldName("type"); typeNode != nil {
			types = append(types, string(typeNode.Content(src)))
		}
	}
	switch {
	case len(types) == 1:
		return classifyGoType(types[0])
	case len(types) == 2 && types[1] == "error":
		return classifyGoType(types[0])
	}
	return nil
}

func extractGoParam(n *sitter.Node, src []byte) []descriptor.ParamDescriptor {
	switch n.Type() {
	case "parameter_declaration":
		// Collect identifier children (names); the last non-identifier named
		// child is the type. One declaration may name multiple params:
		// func f(x, y int)  →  [{x int}, {y int}]
		var names []string
		var typ string
		for i := 0; i < int(n.NamedChildCount()); i++ {
			child := n.NamedChild(i)
			if child.Type() == "identifier" {
				names = append(names, string(child.Content(src)))
			} else {
				typ = string(child.Content(src))
			}
		}
		if len(names) == 0 {
			// Unnamed parameter: func f(int)
			return []descriptor.ParamDescriptor{}
		}
		args := make([]descriptor.ParamDescriptor, len(names))
		for i, name := range names {
			args[i] = descriptor.ParamDescriptor{
				Name: name,
				RawTypeText: typ,
				ResolvedType: classifyGoType(typ),
				Default: nil,
				ExtractorNotes: []string{},
			}
		}
		return args

	case "variadic_parameter_declaration":
		// func f(args ...int)
		// Named children: [identifier("args"), type_identifier("int")]
		// The "..." token is anonymous so it doesn't appear in NamedChild.
		var name, typ string
		for i := 0; i < int(n.NamedChildCount()); i++ {
			child := n.NamedChild(i)
			if child.Type() == "identifier" {
				name = string(child.Content(src))
			} else {
				typ = string(child.Content(src))
			}
		}
		return []descriptor.ParamDescriptor{
			{
				Name: name,
				RawTypeText: "..." + typ,
				ResolvedType: classifyGoType("..." + typ),
				Default: nil,
				ExtractorNotes: []string{},
			},
		}
	}

	return nil
}
