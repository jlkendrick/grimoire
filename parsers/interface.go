package parsers

import (
	types "github.com/jlkendrick/grimoire/types"
)

type LanguageAnalyzer interface {
	ExtractSignature(path_to_function string, function_name string) ([]types.Arg, error)
}