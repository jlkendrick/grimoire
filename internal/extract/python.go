package extract

import (
	"github.com/smacker/go-tree-sitter/python"

	sitter "github.com/smacker/go-tree-sitter"
	descriptor "github.com/jlkendrick/grimoire/internal/descriptor"
)

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
		var name string
		var defaultText string // This is a string because we don't know the type yet (will try to be resolved later)
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
		return []descriptor.ParamDescriptor{{
				Name: name,
				RawTypeText: "",
				ResolvedType: nil,
				Default: defaultText,
				ExtractorNotes: []string{"missing type hint"},
			},
		}

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
		
		var name, typ, defaultText string // default is a string because we don't know the type yet (will try to be resolved later)
		if nameNode != nil {
			name = string(nameNode.Content(src))
		} else {
			return nil
		}
		if typeNode != nil {
			typ = string(typeNode.Content(src))
		}
		if valueNode != nil {
			defaultText = string(valueNode.Content(src))
		}

		return []descriptor.ParamDescriptor{{
			Name: name,
			RawTypeText: typ,
			ResolvedType: &descriptor.TypeInfo{
				Name: typ,
			},
			Default: defaultText,
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
