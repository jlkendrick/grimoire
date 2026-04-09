package cmd

import (
	"fmt"
	"github.com/spf13/cobra"

	config "github.com/jlkendrick/sigil/config"
)

var initCmd = &cobra.Command{
	Use:   "init [config_path]",
	Short: "Parses a config file and generates the sigil.yaml manifest",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		config_path := args[0]
		raw_config, err := config.ParseConfig(config_path)
		if err != nil {
			fmt.Printf("Error parsing config file: %v\n", err)
			return
		}

		config_generator := config.ConfigGenerator{ConfigPath: config_path, Config: raw_config}
		err = config_generator.GenerateManifestYAML()
		if err != nil {
			fmt.Printf("Error generating manifest YAML: %v\n", err)
			return
		}
		
		err = config_generator.WriteManifestYAML()
		if err != nil {
			fmt.Printf("Error writing manifest YAML: %v\n", err)
			return
		}

		fmt.Printf("Manifest YAML generated successfully: %s\n", config_generator.ConfigPath)
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}