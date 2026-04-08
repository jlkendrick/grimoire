/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"os"
	"fmt"

	types "github.com/jlkendrick/sigil/types"

	"github.com/spf13/cobra"
)



// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "sigil",
	Short: "Universal declarative execution framework",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}


func GenerateCommands(config *types.Config) error {

	for _, function := range config.Functions {
		command := &cobra.Command{
			Use: function.Name,
			Run: func(cmd *cobra.Command, args []string) {
				fmt.Println("Running function: ", function.Name)
			},
		}

		for _, arg := range function.Args {
			// Types are cast to the appropriate type in the ParseUserConfig function
			switch arg.Type {
			case "string":
				command.Flags().StringP(arg.Name, "", arg.Default.(string), "")
			case "int":
				command.Flags().IntP(arg.Name, "", arg.Default.(int), "")
			case "bool":
				command.Flags().BoolP(arg.Name, "", arg.Default.(bool), "")
			case "float":
				command.Flags().Float64P(arg.Name, "", arg.Default.(float64), "")
			default:
				return fmt.Errorf("unsupported type: %s", arg.Type)
			}

			if arg.Default != nil {
				command.MarkFlagRequired(arg.Name)
			}
		}

		rootCmd.AddCommand(command)
	}

	return nil
}
// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.sigil.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}


