package cmd

import (
	"fmt"
	"github.com/spf13/cobra"

	config "github.com/jlkendrick/grimoire/config"
)

var add_cmd = &cobra.Command{
	Use:   "add [path_to_function:function_name]",
	Short: "Add a function to the grim.yaml file",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		parts := strings.Split(args[0], ":")
		path_to_function := parts[0]
		function_name := parts[1]


		
		config_path := "sigil.yaml"
		if len(args) > 0 {
			config_path = args[0]
		}
		raw_config, err := config.ParseConfig(config_path)
		if err != nil {
			fmt.Printf("Error parsing config file: %v\n", err)
			return
		}

		config_generator := config.ConfigGenerator{ConfigPath: config_path, Config: raw_config}
		manifest_yaml, err := config_generator.GenerateManifestYAML()
		if err != nil {
			fmt.Printf("Error generating manifest YAML: %v\n", err)
			return
		}
		
		err = config_generator.WriteManifestYAML(manifest_yaml)
		if err != nil {
			fmt.Printf("Error writing manifest YAML: %v\n", err)
			return
		}

		fmt.Printf("Manifest YAML generated successfully: %s\n", config_generator.ConfigPath)
	},
}

func init() {
	add_cmd.Flags().BoolP("global", "g", false, "Add the function to the global grimoire")
	rootCmd.AddCommand(add_cmd)
}