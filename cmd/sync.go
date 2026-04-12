package cmd

import (
	"fmt"


	"github.com/spf13/cobra"
)

var sync_cmd = &cobra.Command{
	Use:   "sync",
	Short: "Automatically generate arguments for all functions in the grim.yaml file",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Automatically generating arguments for all functions in the grim.yaml file...")

		global, err := cmd.Flags().GetBool("global")
		if err != nil {
			fmt.Printf("Error getting global flag: %v\n", err)
			return
		}

		if global {
			fmt.Println("Syncing global grimoire...")
		}
	},
}

func init() {
	sync_cmd.Flags().BoolP("global", "g", false, "Sync the global grimoire")
	rootCmd.AddCommand(sync_cmd)
}