package parsers

import (
	types "github.com/jlkendrick/grimoire/types"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
)

var goConfig = grammarConfig{
	language:         golang.GetLanguage,
	functionNodeType: "function_declaration",
	parametersField:  "parameters",
	extractParam:     extractGoParam,
}

type GoAnalyzer struct{}

func (a *GoAnalyzer) ExtractSignature(abs_path_to_function, funcName string) ([]types.Arg, error) {
	return extractSignatureBase(goConfig, abs_path_to_function, funcName)
}

func extractGoParam(n *sitter.Node, src []byte) []types.Arg {
	switch n.Type() {
	case "parameter_declaration":
		// Collect identifier children (names); the last non-identifier named
		// child is the type. One declaration may name multiple params:
		//   func f(x, y int)  →  [{x int}, {y int}]
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
			return []types.Arg{{Type: typ}}
		}
		args := make([]types.Arg, len(names))
		for i, name := range names {
			args[i] = types.Arg{Name: name, Type: typ}
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
		return []types.Arg{{Name: name, Type: "..." + typ}}
	}

	return nil
}
