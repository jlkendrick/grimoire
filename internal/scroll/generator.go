package scroll

import (
	"fmt"
	"strings"

	extract "github.com/jlkendrick/grimoire/internal/extract"
	descriptor "github.com/jlkendrick/grimoire/internal/descriptor"
)

type SpellGenerator struct {
	AbsPathToFunction string
	FunctionName   	  string
}

func (g *SpellGenerator) GenerateDescriptor() (descriptor.FunctionDescriptor, error) {
	var extractor extract.LanguageExtractor

	if !strings.Contains(g.AbsPathToFunction, ".") {
		return descriptor.FunctionDescriptor{}, fmt.Errorf("no file extension found: %s", g.AbsPathToFunction)
	}

	// Determine the file extension and use the appropriate analyzer
	file_extensions := strings.Split(g.AbsPathToFunction, ".")
	file_extension := file_extensions[len(file_extensions)-1]
	switch file_extension {
	case "py":
		extractor = &extract.PythonExtractor{}
	case "go":
		extractor = &extract.GoExtractor{} // TODO
	default:
		return descriptor.FunctionDescriptor{}, fmt.Errorf("unsupported file extension: %s", file_extension)
	}

	function_descriptor, err := extractor.DescribeFunction(g.AbsPathToFunction, g.FunctionName)
	if err != nil {
		return descriptor.FunctionDescriptor{}, err
	}

	return function_descriptor, nil
}

func MinifySpellDescriptor(spell_descriptor descriptor.FunctionDescriptor) Spell {
	return Spell{
		Command: spell_descriptor.CommandName,
		Path: spell_descriptor.SourceFile,
		Function: spell_descriptor.FunctionName,
	}
}