package cmd

import (
	"os"
	"fmt"
	"path/filepath"

	core "github.com/jlkendrick/grimoire/core"
	utils "github.com/jlkendrick/grimoire/utils"

	"github.com/spf13/cobra"
)

var clean_cmd = &cobra.Command{
	Use:   "clean [global]",
	Short: "Clean venvs for unused functions in existing spells",
	Run: func(cmd *cobra.Command, args []string) {
		global, err := cmd.Flags().GetBool("global")
		if err != nil {
			fmt.Printf("Error getting global flag: %v\n", err)
			return
		}
		force_clean, err := cmd.Flags().GetBool("force")
		if err != nil {
			fmt.Printf("Error getting force flag: %v\n", err)
			return
		}
		if force_clean {
			// Delete all venvs in the .grimoire/envs directory
			grimoire_home, err := utils.GrimoireHome()
			if err != nil {
				fmt.Printf("Error resolving grimoire home: %v\n", err)
				return
			}
			envs_dir := filepath.Join(grimoire_home, "envs")
			envs, err := os.ReadDir(envs_dir)
			if err != nil {
				fmt.Printf("Error reading envs directory: %v\n", err)
				return
			}
			for _, env := range envs {
				os.RemoveAll(filepath.Join(envs_dir, env.Name()))
			}
			return
		}

		var config_type string
		if global {
			config_type = "global"
		} else {
			config_type = "local"
		}
		config, err := core.LoadConfig(config_type)
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			return
		}
		// For each function, reconstruct the path to the function file and check if it exists
		unused_functions := map[string]bool{}
		for _, function := range config.Functions {
			function_path := filepath.Join(filepath.Dir(function.SpellPath), function.TargetFile)
			if _, err := os.Stat(function_path); os.IsNotExist(err) {
				unused_functions[function_path] = true
			}
		}

		// Go through each venv and check if it is in the unused_functions map
		grimoire_home, err := utils.GrimoireHome()
		if err != nil {
			fmt.Printf("Error resolving grimoire home: %v\n", err)
			return
		}
		venv_root := filepath.Join(grimoire_home, "envs")
		venv_paths, err := os.ReadDir(venv_root)
		if err != nil {
			fmt.Printf("Error reading venvs: %v\n", err)
			return
		}
		deleted_venvs := 0
		for _, venv := range venv_paths {
			if venv.IsDir() {
				// Get the origin spell path from the .grimoire_origin file
				origin_pointer_file := filepath.Join(venv_root, venv.Name(), ".grimoire_origin")
				origin_pointer_file_content, err := os.ReadFile(origin_pointer_file)
				if err != nil {
					fmt.Printf("Error reading origin pointer file: %v\n", err)
					return
				}
				origin_spell_path := string(origin_pointer_file_content)

				// If the origin spell path is in the unused_functions map, delete the venv
				if _, ok := unused_functions[origin_spell_path]; ok {
					err = os.RemoveAll(filepath.Join(venv_root, venv.Name()))
					if err != nil {
						fmt.Printf("Error deleting venv: %v\n", err)
						return
					}
					deleted_venvs++
				}
			}
		}

		fmt.Printf("Deleted %d unused venvs\n", deleted_venvs)
	},
}

func init() {
	clean_cmd.Flags().BoolP("global", "g", false, "Clean global venvs")
	clean_cmd.Flags().BoolP("force", "f", false, "Force clean all venvs")
	rootCmd.AddCommand(clean_cmd)
}