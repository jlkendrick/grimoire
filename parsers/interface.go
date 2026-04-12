package parsers

import (
	types "github.com/jlkendrick/grimoire/types"
)

type LanguageAnalyzer interface {
	ExtractSignature(function types.Function) ([]types.Arg, error)
}