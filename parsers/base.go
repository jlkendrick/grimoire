package parsers

import (
	"context"
	"fmt"
	"os"

	types "github.com/jlkendrick/grimoire/types"
	sitter "github.com/smacker/go-tree-sitter"
)

type LanguageAnalyzer interface {
	ExtractSignature(path_to_function string, function_name string) ([]types.Arg, error)
}

// grammarConfig holds the language-specific knobs needed to extract a
// function signature. The pipeline itself (file I/O, parsing, tree
// traversal, parameter accumulation) lives in extractSignatureBase and is
// shared by every LanguageAnalyzer implementation.
type grammarConfig struct {
	// language returns the tree-sitter grammar to use.
	language func() *sitter.Language

	// functionNodeType is the AST node type that represents a function
	// definition in this grammar (e.g. "function_definition" for Python,
	// "function_declaration" for Go/JS).
	functionNodeType string

	// parametersField is the field name used to reach the parameter list
	// on the function node (almost always "parameters").
	parametersField string

	// extractParam converts a single parameter node into an Arg.
	// Return (Arg{}, false) to skip unsupported node kinds.
	extractParam func(n *sitter.Node, src []byte) (types.Arg, bool)
}

func extractSignatureBase(cfg grammarConfig, path, funcName string) ([]types.Arg, error) {
	parser := sitter.NewParser()
	parser.SetLanguage(cfg.language())

	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	tree, err := parser.ParseCtx(context.Background(), nil, src)
	if err != nil {
		return nil, err
	}

	fnNode := findFunctionNode(tree.RootNode(), src, funcName, cfg.functionNodeType)
	if fnNode == nil {
		return nil, fmt.Errorf("function %s not found in %s", funcName, path)
	}

	paramsNode := fnNode.ChildByFieldName(cfg.parametersField)
	if paramsNode == nil {
		return []types.Arg{}, nil
	}

	args := []types.Arg{}
	for i := 0; i < int(paramsNode.NamedChildCount()); i++ {
		paramNode := paramsNode.NamedChild(i)
		if paramNode == nil {
			continue
		}
		if arg, ok := cfg.extractParam(paramNode, src); ok {
			args = append(args, arg)
		}
	}

	return args, nil
}

// findFunctionNode performs a DFS over the AST looking for a node of
// functionNodeType whose "name" field (or any identifier child) matches
// funcName.
func findFunctionNode(root *sitter.Node, src []byte, funcName, functionNodeType string) *sitter.Node {
	var dfs func(*sitter.Node) *sitter.Node
	dfs = func(node *sitter.Node) *sitter.Node {
		if node == nil {
			return nil
		}

		if node.Type() == functionNodeType {
			if nameNode := node.ChildByFieldName("name"); nameNode != nil {
				if string(nameNode.Content(src)) == funcName {
					return node
				}
			}
			// Fallback: scan named children for an identifier with the right name.
			for i := 0; i < int(node.NamedChildCount()); i++ {
				child := node.NamedChild(i)
				if child != nil && child.Type() == "identifier" && string(child.Content(src)) == funcName {
					return node
				}
			}
		}

		for i := 0; i < int(node.NamedChildCount()); i++ {
			if result := dfs(node.NamedChild(i)); result != nil {
				return result
			}
		}
		return nil
	}
	return dfs(root)
}
