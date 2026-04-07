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

	err = functions[0].InferArgs()
	if err != nil {
		fmt.Println("Error inferring arguments:", err)
		return
	}

	for _, function := range functions {
		fmt.Println(function)
	}

	fmt.Println("Functions parsed successfully")
	fmt.Println("Simulating running hello_world_func")

	arg_map := map[string]any{
		"n": "hello",
	}
	function := functions[0] // hello_world_func
	err = function.ValidateArgs(arg_map)
	if err != nil {
		fmt.Println("Error validating arguments:", err)
		return
	}
	fmt.Println("Arguments validated successfully")
	fmt.Println("Running function:", function.Name)
	

}