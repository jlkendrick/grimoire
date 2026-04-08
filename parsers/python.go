package parsers

import (
	config "github.com/jlkendrick/janus/config"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/python"
)

func ExtractSignature(function config.Function) ([]config.Arg, error) {
	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())
	
	source_code := []byte(function.LoadSourceCode())
	

	return nil, nil
}