package parsers

import (
	types "github.com/jlkendrick/sigil/types"
)

type LanguageAnalyzer interface {
	ExtractSignature(function types.Function) ([]types.Arg, error)
}