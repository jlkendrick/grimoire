package weave

import (
	"fmt"
	"os"

	scroll "github.com/jlkendrick/grimoire/internal/scroll"
	parser "github.com/jlkendrick/grimoire/weave/parser"
)

// WeaveFile reads a .wv source file, parses it, and transpiles it into a
// scroll.Ritual. It does NOT validate against a descriptor cache or write
// anything to disk — those are the caller's responsibility so the CLI layer
// can sequence parse → transpile → validate → emit with appropriate error
// reporting at each step.
func WeaveFile(path string) (scroll.Ritual, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return scroll.Ritual{}, fmt.Errorf("reading %s: %w", path, err)
	}

	ast_ritual, err := parser.Parse(string(content))
	if err != nil {
		return scroll.Ritual{}, fmt.Errorf("parsing %s: %w", path, err)
	}

	tr := Transpiler{}
	ritual, err := tr.TranspileRitual(ast_ritual)
	if err != nil {
		return scroll.Ritual{}, fmt.Errorf("transpiling %s: %w", path, err)
	}

	return ritual, nil
}
