package parser

import (
	"github.com/alecthomas/participle/v2"
)

var GrimoireParser = participle.MustBuild[Ritual](
	participle.UseLookahead(2),
)

func Parse(input string) (*Ritual, error) {
	return GrimoireParser.ParseString("", input)
}