package parsers

import (
	types "github.com/jlkendrick/grimoire/types"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/python"
)

var pythonConfig = grammarConfig{
	language:         python.GetLanguage,
	functionNodeType: "function_definition",
	parametersField:  "parameters",
	extractParam:     extractPythonParam,
}

type PythonAnalyzer struct{}

func (a *PythonAnalyzer) ExtractSignature(path, funcName string) ([]types.Arg, error) {
	return extractSignatureBase(pythonConfig, path, funcName)
}

func extractPythonParam(n *sitter.Node, src []byte) []types.Arg {
	switch n.Type() {
	case "identifier":
		// def f(x):
		return []types.Arg{{Name: string(n.Content(src))}}

	case "default_parameter":
		// def f(x=1):
		var name, defaultText string
		for i := 0; i < int(n.NamedChildCount()); i++ {
			child := n.NamedChild(i)
			if child.Type() == "identifier" {
				name = string(child.Content(src))
			} else {
				defaultText = string(child.Content(src))
			}
		}
		if name == "" {
			return nil
		}
		return []types.Arg{{Name: name, Default: defaultText}}

	case "typed_parameter":
		// def f(x: int):
		var name, typ string
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
		return []types.Arg{{Name: name, Type: typ}}

	case "typed_default_parameter":
		// def f(x: int = 1):
		nameNode := n.ChildByFieldName("name")
		typeNode := n.ChildByFieldName("type")
		valueNode := n.ChildByFieldName("value")
		if nameNode == nil {
			return nil
		}
		arg := types.Arg{Name: string(nameNode.Content(src))}
		if typeNode != nil {
			arg.Type = string(typeNode.Content(src))
		}
		if valueNode != nil {
			arg.Default = string(valueNode.Content(src))
		}
		return []types.Arg{arg}

	case "list_splat_pattern": // *args
		if n.NamedChildCount() == 0 {
			return nil
		}
		return []types.Arg{{Name: "*" + string(n.NamedChild(0).Content(src))}}

	case "dictionary_splat_pattern": // **kwargs
		if n.NamedChildCount() == 0 {
			return nil
		}
		return []types.Arg{{Name: "**" + string(n.NamedChild(0).Content(src))}}
	}

	// Unsupported param kind (pos-only marker '/', etc.)
	return nil
}
