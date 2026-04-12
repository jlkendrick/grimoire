package core

import (
	"os"

	types "github.com/jlkendrick/grimoire/types"
	utils "github.com/jlkendrick/grimoire/utils"
	config "github.com/jlkendrick/grimoire/config"
)

var cached_config *types.Config

func LoadConfig() (*types.Config, error) {
	if cached_config != nil {
		return cached_config, nil
	}
	
	current_dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	// Determine the path to the config file
	var config_path string
	matched_targets, found := utils.UpwardsTraversalForTargets(current_dir, []string{"grim.yaml"})
	if found {
		config_path = matched_targets["grim.yaml"]
	} else {
		// Default to the global grimoire
		config_path, err = utils.ExpandUserPath("~/Code/Projects/grimoire/.grimoire/config.yaml") // UPDATE LATER WITH PERMANENT CONFIG FILE PATH
		if err != nil {
			return nil, err
		}
	}

	// Parse the config file
	config, err := config.ParseConfig(config_path)
	if err != nil {
		return nil, err
	}

	// Cache the config and return
	cached_config = config
	return config, nil
}