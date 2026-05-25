package weave

import (
	"fmt"
	"os"
	"path/filepath"

	parser "github.com/jlkendrick/grimoire/weave/parser"
)

func TestParse() {
	// Load the test file
	srcPath := filepath.Join(os.Getenv("HOME"), "Code/Projects/grimoire/sample/test.grm")
	content, err := os.ReadFile(srcPath)
	if err != nil {
		fmt.Printf("Failed to read file: %v", err)
	}

	// Parse the file
	ast_ritual, err := parser.Parse(string(content))
	if err != nil {
		fmt.Printf("Failed to parse file: %v", err)
	}

	// Print the ritual
	fmt.Println(ast_ritual)

	// Transpile the ast ritual into a scroll.Ritual
	transpiler := Transpiler{}
	ritual, err := transpiler.transpileRitual(ast_ritual)
	if err != nil {
		fmt.Printf("Failed to lower ritual: %v", err)
	}

	for i, step := range ritual.Steps {
		fmt.Println(i + 1, ":", step)
	}
}