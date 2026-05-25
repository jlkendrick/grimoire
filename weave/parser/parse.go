package parser

import (
	"github.com/alecthomas/participle/v2"
)

var GrimoireParser = participle.MustBuild[Ritual](
	// Lookahead must reach past `Ident '.' Ident` so CallExpr's optional
	// module qualifier can backtrack into a RefExpr when no `(` follows
	// (e.g. `ref.field` as an argument value).
	participle.UseLookahead(3),
)

func Parse(input string) (*Ritual, error) {
	return GrimoireParser.ParseString("", input)
}