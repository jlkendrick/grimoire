package cmd

import (
	"os"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	cache "github.com/jlkendrick/grimoire/internal/cache"
	utils "github.com/jlkendrick/grimoire/internal/utils"
	scroll "github.com/jlkendrick/grimoire/internal/scroll"
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
			fmt.Printf("%s Purged all conjured environments\n", utils.AccentStyle("+"))
			return
		}

		// Find the scrolls that we need to check for orphaned spells
		var scrolls []*scroll.Scroll
		if global {
			scrolls, err = scroll.LoadGlobalScrolls()
			if err != nil {
				fmt.Printf("Error loading global scrolls: %v\n", err)
				return
			}
		} else {
			scrolls, err = scroll.LoadScrolls()
			if err != nil {
				fmt.Printf("Error loading local scroll: %v\n", err)
				return
			}
		}

		var scroll_paths []string
		for _, scroll := range scrolls {
			scroll_paths = append(scroll_paths, scroll.Path)
		}
		
		// For each spell in each scroll, reconstruct the path to the function file and check if it exists
		unused_functions := map[string]bool{}
		for _, scroll := range scrolls {
			// Read the descriptor cache for the scroll
			descriptor_cache, err := cache.ReadDescriptorCache(scroll.Path)
			if err != nil {
				fmt.Printf("Error reading descriptor cache: %v\n", err)
				return
			}
			for _, function := range descriptor_cache.Functions {
				if _, err := os.Stat(function.AbsPathToSourceFile); os.IsNotExist(err) {
					unused_functions[function.AbsPathToSourceFile] = true
				}
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
			if !venv.IsDir() {
				continue
			}
			env_dir := filepath.Join(venv_root, venv.Name())

			// Go envs carry a wrapper go.mod at root and per-spell subdirs,
			// each with its own .grimoire_origin. Orphan detection runs per
			// spell subdir, and the env itself is removed only when nothing
			// alive remains.
			if _, err := os.Stat(filepath.Join(env_dir, "go.mod")); err == nil {
				spell_dirs, err := os.ReadDir(env_dir)
				if err != nil {
					fmt.Printf("Error reading env dir: %v\n", err)
					return
				}
				remaining := 0
				for _, sd := range spell_dirs {
					if !sd.IsDir() {
						continue
					}
					spell_path := filepath.Join(env_dir, sd.Name())
					origin_content, err := os.ReadFile(filepath.Join(spell_path, ".grimoire_origin"))
					if err != nil {
						continue
					}
					if _, ok := unused_functions[string(origin_content)]; ok {
						if err := os.RemoveAll(spell_path); err != nil {
							fmt.Printf("Error deleting spell dir: %v\n", err)
							return
						}
						deleted_venvs++
					} else {
						remaining++
					}
				}
				if remaining == 0 {
					if err := os.RemoveAll(env_dir); err != nil {
						fmt.Printf("Error deleting env: %v\n", err)
						return
					}
				}
				continue
			}

			// Python-style env: a single .grimoire_origin at the venv root.
			origin_pointer_file := filepath.Join(env_dir, ".grimoire_origin")
			origin_pointer_file_content, err := os.ReadFile(origin_pointer_file)
			if err != nil {
				fmt.Printf("Error reading origin pointer file: %v\n", err)
				return
			}
			origin_scroll_path := string(origin_pointer_file_content)
			if _, ok := unused_functions[origin_scroll_path]; ok {
				if err := os.RemoveAll(env_dir); err != nil {
					fmt.Printf("Error deleting venv: %v\n", err)
					return
				}
				deleted_venvs++
			}
		}

		if deleted_venvs == 0 {
			fmt.Printf("%s No dormant environments to dispel\n", utils.AccentStyle("+"))
		} else {
			fmt.Printf("%s Dispelled %d dormant environments\n", utils.AccentStyle("+"), deleted_venvs)
		}
	},
}

func init() {
	clean_cmd.Flags().BoolP("global", "g", false, "Clean global venvs")
	clean_cmd.Flags().BoolP("force", "f", false, "Force clean all venvs")
	rootCmd.AddCommand(clean_cmd)
}