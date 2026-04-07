package main

import (
	"fmt"
	"github.com/jlkendrick/janus/parsers"
)

func main() {

	functions, err := parsers.ParseYAML("janus.yaml")
	if err != nil {
		fmt.Println("Error parsing YAML:", err)
		return
	}

	for _, function := range functions {
		fmt.Println(function)
	}
}