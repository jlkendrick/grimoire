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
}

type GoExtractor struct{}

func (a *GoExtractor) GenerateDescriptor_ParamsOnly(abs_path_to_function, funcName string) (descriptor.FunctionDescriptor, error) {
	return generateDescriptorBase_ParamsOnly(goConfig, abs_path_to_function, funcName)
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
