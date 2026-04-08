package main

import (
	"fmt"

	config "github.com/jlkendrick/janus/config"
)

func main() {
	config_path := "janus.yaml"

	// Parse the user's configuration file
	// This will contain basic information about the functions we want to support
	user_config, err := config.ParseUserConfig(config_path)
	if err != nil {
		fmt.Println("Error parsing YAML:", err)
		return
	}

	// By default, we will automatically generate the 'args' field for the user's functions
	// This will later be used to validate the arguments passed to the function
	generator := config.ConfigGenerator{
		ConfigPath: config_path,
		Config:     user_config,
	}
	err = generator.GenerateTypedYAML()
	if err != nil {
		fmt.Println("Error generating typed YAML:", err)
		return
	}

	for _, function := range user_config.Functions {
		fmt.Println(function)
	}

	fmt.Println("Functions parsed successfully")
	fmt.Println("Simulating running hello_world_func")
	
}