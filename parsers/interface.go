package parsers

import (
	types "github.com/jlkendrick/janus/types"
)

type LanguageAnalyzer interface {
	ExtractSignature(function types.Function) ([]types.Arg, error)
}