package core

import (
	"os"
	"fmt"

	types "github.com/jlkendrick/grimoire/types"
	utils "github.com/jlkendrick/grimoire/utils"
	config "github.com/jlkendrick/grimoire/config"
)

var cached_local_config *types.Config
var cached_global_config *types.Config

func LoadConfig(config_type string) (*types.Config, string, error) {
	switch config_type {
	case "local":	
		if cached_local_config != nil {
			return cached_local_config, "", nil
		}

		fmt.Println("[DEBUG] Reading local config...")
		
		current_dir, err := os.Getwd()
		if err != nil {
			return nil, "", err
		}

		// Determine the path to the config file
		var config_path string
		var defaulted_to_global bool
		matched_targets, found := utils.UpwardsTraversalForTargets(current_dir, []string{"grim.yaml"})
		if found {
			config_path = matched_targets["grim.yaml"]
			defaulted_to_global = false
		} else {
			// Default to the global grimoire
			config_path, err = utils.ExpandUserPath("~/Code/Projects/grimoire/.grimoire/config.yaml") // UPDATE LATER WITH PERMANENT CONFIG FILE PATH
			if err != nil {
				return nil, "", err
			}
			defaulted_to_global = true
		}

		// Parse the config file
		config, err := config.ParseConfig(config_path)
		if err != nil {
			return nil, "", err
		}

		// Cache the config and return
		if defaulted_to_global {
			cached_global_config = config
		} else {
			cached_local_config = config
		}
		return config, config_path, nil
	
	case "global":
		if cached_global_config != nil {
			return cached_global_config, "", nil
		}

		fmt.Println("[DEBUG] Reading global config...")

		config_path, err := utils.ExpandUserPath("~/Code/Projects/grimoire/.grimoire/config.yaml") // UPDATE LATER WITH PERMANENT CONFIG FILE PATH
		if err != nil {
			return nil, "", err
		}

		config, err := config.ParseConfig(config_path)
		if err != nil {
			return nil, "", err
		}

		// Cache the config and return
		cached_global_config = config
		return config, config_path, nil

	default:
		return nil, "", fmt.Errorf("invalid config type: %s", config_type)
	}
}