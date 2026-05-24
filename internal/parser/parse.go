package parser

import (
	"os"
	"fmt"
	"path/filepath"

	"github.com/alecthomas/participle/v2"
)

var GrimoireParser = participle.MustBuild[Ritual](
	participle.UseLookahead(2),
)

func Parse(input string) (*Ritual, error) {
	return GrimoireParser.ParseString("", input)
}

func TestParse() {
	// Load the test file
	srcPath := filepath.Join(os.Getenv("HOME"), "Code/Projects/grimoire/sample/test.grm")
	content, err := os.ReadFile(srcPath)
	if err != nil {
		fmt.Printf("Failed to read file: %v", err)
	}

	// Parse the file
	ritual, err := Parse(string(content))
	if err != nil {
		fmt.Printf("Failed to parse file: %v", err)
	}

	// Print the ritual
	fmt.Println(ritual)

	// Lower the ritual
	transpiler := Transpiler{}
	yaml_steps, err := transpiler.transpileSteps(ritual.Body)
	if err != nil {
		fmt.Printf("Failed to lower ritual: %v", err)
	}

	for i, step := range yaml_steps {
		fmt.Println(i + 1, ":", step)
	}
}