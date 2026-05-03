package extract

import (
	"os"
	"fmt"
	"context"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"

	descriptor "github.com/jlkendrick/grimoire/internal/descriptor"
	scroll "github.com/jlkendrick/grimoire/internal/scroll"
)

type FunctionDescriptorGenerator struct {
	AbsPathToFunction string
	FunctionName   	  string
}

func (g *FunctionDescriptorGenerator) GenerateDescriptor() (descriptor.FunctionDescriptor, error) {
	var extractor LanguageExtractor

	if !strings.Contains(g.AbsPathToFunction, ".") {
		return descriptor.FunctionDescriptor{}, fmt.Errorf("no file extension found: %s", g.AbsPathToFunction)
	}

	// Determine the file extension and use the appropriate analyzer
	file_extensions := strings.Split(g.AbsPathToFunction, ".")
	file_extension := file_extensions[len(file_extensions)-1]
	switch file_extension {
	case "py":
		extractor = &PythonExtractor{}
	case "go":
		extractor = &GoExtractor{} // TODO
	default:
		return descriptor.FunctionDescriptor{}, fmt.Errorf("unsupported file extension: %s", file_extension)
	}

	function_descriptor, err := extractor.DescribeFunction(g.AbsPathToFunction, g.FunctionName)
	if err != nil {
		return descriptor.FunctionDescriptor{}, err
	}

	return function_descriptor, nil
}

func MinifyFunctionDescriptor(function_descriptor descriptor.FunctionDescriptor) scroll.Spell {
	return scroll.Spell{
		Command: function_descriptor.CommandName,
		Path: function_descriptor.SourceFile,
		Function: function_descriptor.FunctionName,
	}
}

type LanguageExtractor interface {
	DescribeFunction(abs_path_to_function string, function_name string) (descriptor.FunctionDescriptor, error)
}

// grammarConfig holds the language-specific knobs needed to extract a
// function signature. The pipeline itself (file I/O, parsing, tree
// traversal, parameter accumulation) lives in extractSignatureBase and is
// shared by every LanguageExtractor implementation.
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

	// extractParam converts a single parameter node into zero or more Args.
	// Return nil to skip unsupported node kinds (e.g. position-only markers).
	// Returning multiple Args handles languages like Go where one declaration
	// can name several parameters sharing a type: func f(x, y int).
	extractParam func(n *sitter.Node, src []byte) []descriptor.ParamDescriptor
}

func describeFunctionBase(cfg grammarConfig, path, funcName string) (descriptor.FunctionDescriptor, error) {
	var function_descriptor descriptor.FunctionDescriptor

	// Fill in the easy stuff
	function_descriptor.CommandName = funcName
	function_descriptor.SourceFile = path

	// Extract the function signature using our method of choice (determined in ExtractParams)
	params, err := extractParamsBase(cfg, path, funcName)
	if err != nil {
		return descriptor.FunctionDescriptor{}, err
	}

	// Fill in the params
	function_descriptor.Params = params

	return function_descriptor, nil
}

func extractParamsBase(cfg grammarConfig, path, funcName string) ([]descriptor.ParamDescriptor, error) {
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
		return []descriptor.ParamDescriptor{}, nil
	}

	params := []descriptor.ParamDescriptor{}
	for i := 0; i < int(paramsNode.NamedChildCount()); i++ {
		paramNode := paramsNode.NamedChild(i)
		if paramNode == nil {
			continue
		}
		params = append(params, cfg.extractParam(paramNode, src)...)
	}

	return params, nil
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

