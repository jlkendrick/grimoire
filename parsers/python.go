package parsers

import (
	"fmt"
	"context"

	types "github.com/jlkendrick/grimoire/types"
	sitter "github.com/smacker/go-tree-sitter"

	"github.com/smacker/go-tree-sitter/python"
)


type PythonAnalyzer struct {}

// Extract and parse the signature of a function into Args
func (a *PythonAnalyzer) ExtractSignature(function types.Function) ([]types.Arg, error) {
	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())
	
	source_code, err := function.LoadSourceCode()
	if err != nil {
		return nil, err
	}
	
	// Parse the source code into a tree
	tree, err := parser.ParseCtx(context.Background(), nil, source_code)
	if err != nil {
		return nil, err
	}
	root := tree.RootNode()
	
	// Find the function node
	fn_node := findFunctionNode(root, source_code, function.TargetFunction)
	if fn_node == nil {
		return nil, fmt.Errorf("function %s not found in %s", function.TargetFunction, function.TargetFile)
	}

	// Extract the function signature from the function node
	params_node := fn_node.ChildByFieldName("parameters")
	if params_node == nil {
		return []types.Arg{}, nil
	}

	args := []types.Arg{}
	for i := 0; i < int(params_node.ChildCount()); i++ {
		param_node := params_node.NamedChild(i)
		if param_node == nil {
			continue
		}

		arg, ok := extractArgFromParamNode(param_node, source_code)
		if ok {
			args = append(args, arg)
		}
	}

	return args, nil
}

func findFunctionNode(root *sitter.Node, source_code []byte, function_name string) *sitter.Node {
	var dfs func(node *sitter.Node) *sitter.Node

	dfs = func(node *sitter.Node) *sitter.Node {
		if node == nil {
			return nil
		}

		if node.Type() == "function_definition" {
			
			name_node := node.ChildByFieldName("name")
			if name_node != nil && string(name_node.Content(source_code)) == function_name {
				return node
			}

			// Fallback to searching named children
			for i := 0; i < int(node.NamedChildCount()); i++ {
				child := node.NamedChild(i)
				if child != nil && child.Type() == "identifier" && string(child.Content(source_code)) == function_name {
					return node
				}
			}
		}

		// Recurse through children
		for i := 0; i < int(node.NamedChildCount()); i++ {
			if child_node := dfs(node.NamedChild(i)); child_node != nil {
				return child_node
			}
		}

		return nil
	}

	return dfs(root)
}

func extractArgFromParamNode(n *sitter.Node, source_code []byte) (types.Arg, bool) {
	switch n.Type() {
	case "identifier":
			// def f(x):
			name := string(n.Content(source_code))
			return types.Arg{
					Name:    name,
					Type:    "",
					Default: nil,
			}, true

	case "default_parameter":
			// def f(x=1):
			// Children: identifier, "=", expression
			var name string
			var defaultText string

			for i := 0; i < int(n.NamedChildCount()); i++ {
					child := n.NamedChild(i)
					switch child.Type() {
					case "identifier":
							name = string(child.Content(source_code))
					default:
							// Treat the other named child after '=' as the default expr
							defaultText = string(child.Content(source_code))
					}
			}

			if name == "" {
					return types.Arg{}, false
			}

			return types.Arg{
					Name:    name,
					Type:    "",
					Default: defaultText,
			}, true

	case "typed_parameter":
			// def f(x: int):
			// First named child is the identifier; remaining named child is the type.
			var name, typ string
			for i := 0; i < int(n.NamedChildCount()); i++ {
					child := n.NamedChild(i)
					if child == nil {
							continue
					}
					if child.Type() == "identifier" {
							name = string(child.Content(source_code))
					} else {
							typ = string(child.Content(source_code))
					}
			}
			if name == "" {
					return types.Arg{}, false
			}
			return types.Arg{
					Name:    name,
					Type:    typ,
					Default: nil,
			}, true

	case "typed_default_parameter":
			// def f(x: int = 1):
			nameNode := n.ChildByFieldName("name")
			typeNode := n.ChildByFieldName("type")
			valueNode := n.ChildByFieldName("value") // depends on grammar version

			if nameNode == nil {
					return types.Arg{}, false
			}

			name := string(nameNode.Content(source_code))
			var typ string
			if typeNode != nil {
					typ = string(typeNode.Content(source_code))
			}

			var defaultText any
			if valueNode != nil {
					defaultText = string(valueNode.Content(source_code))
			}

			return types.Arg{
					Name:    name,
					Type:    typ,
					Default: defaultText,
			}, true

	case "list_splat_pattern": // *args
			if n.NamedChildCount() == 0 {
					return types.Arg{}, false
			}
			ident := n.NamedChild(0)
			return types.Arg{
					Name:    "*" + string(ident.Content(source_code)),
					Type:    "",
					Default: nil,
			}, true

	case "dictionary_splat_pattern": // **kwargs
			if n.NamedChildCount() == 0 {
					return types.Arg{}, false
			}
			ident := n.NamedChild(0)
			return types.Arg{
					Name:    "**" + string(ident.Content(source_code)),
					Type:    "",
					Default: nil,
			}, true
	}

	// Unsupported param kind (Pos-only marker '/', etc.)
	return types.Arg{}, false
}