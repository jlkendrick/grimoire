package main

import (
	"fmt"

	config "github.com/jlkendrick/sigil/config"
	cmd "github.com/jlkendrick/sigil/cmd"
)

func main() {

	// Parse the user's configuration file
	// This will contain basic information about the functions we want to support
	config_path := "sigil.yaml"
	user_config, err := config.ParseUserConfig(config_path)
	if err != nil {
		fmt.Println("Error parsing YAML:", err)
		return
	}

	err = cmd.GenerateCommands(user_config)
	if err != nil {
		fmt.Println("Error generating commands:", err)
		return
	}

	cmd.Execute()
}