package main

import (
	"fmt"

	config "github.com/jlkendrick/janus/config"
)

func main() {
	config_path := "janus.yaml"

	// Parse the user's configuration file
	// This will contain basic information about the functions we want to support
	functions, err := config.ParseUserYAML(config_path)
	if err != nil {
		fmt.Println("Error parsing YAML:", err)
		return
	}

	// By default, we will automatically generate the 'args' field for the user's functions
	// This will later be used to validate the arguments passed to the function
	generator := config.ConfigGenerator{
		ConfigPath: config_path,
		Functions: functions,
	}
	err = generator.GenerateTypedYAML()

	for _, function := range functions {
		fmt.Println(function)
	}

	fmt.Println("Functions parsed successfully")
	fmt.Println("Simulating running hello_world_func")
	
}