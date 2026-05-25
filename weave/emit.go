package weave

import (
	"fmt"
	"os"
	"path/filepath"

	parser "github.com/jlkendrick/grimoire/weave/parser"
	scroll "github.com/jlkendrick/grimoire/internal/scroll"
)

func TestParse() {
	// Load the test file
	srcPath := filepath.Join(os.Getenv("HOME"), "Code/Projects/grimoire/sample/test.wv")
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

	// Write the ritual to the scroll
	err = transpiler.writeToScroll(ritual)
	if err != nil {
		fmt.Printf("Failed to write ritual to scroll: %v", err)
	}
}

func (t *Transpiler) writeToScroll(ritual scroll.Ritual) error {
	scroll, err := scroll.ResolveAddScroll()
	if err != nil {
		return err
	}

	// Add the ritual to the scroll
	scroll.Rituals = append(scroll.Rituals, ritual)

	// Write the scroll to the file system
	return scroll.Write()
}