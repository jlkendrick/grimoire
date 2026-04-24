package core

import (
	"os"
	"fmt"

	types "github.com/jlkendrick/grimoire/types"
	utils "github.com/jlkendrick/grimoire/utils"
	config "github.com/jlkendrick/grimoire/config"
)

var cached_config *types.Config
var cached_config_path string

// ResetConfigCache clears all cached config state. Intended for use in tests.
func ResetConfigCache() {
	cached_config = nil
	cached_config_path = ""
}

func LoadConfig(config_type string) (*types.Config, error) {
	if cached_config != nil {
		return cached_config, nil
	}

	switch config_type {
	case "local":
		current_dir, err := os.Getwd()
		if err != nil {
			return nil, err
		}

		// Determine the path to the config file
		var config_path string
		matched_targets, found := utils.UpwardsTraversalForTargets(current_dir, []string{"scroll.yaml"})
		if found {
			config_path = matched_targets["scroll.yaml"]
		} else {
			// Fall back to the global grimoire config
			grimoire_home, err := utils.GrimoireHome()
			if err != nil {
				return nil, err
			}
			config_path = grimoire_home + "/grimoire.yaml"
		}

		// Parse the config file
		config, err := config.ParseConfig(config_path)
		if err != nil {
			return nil, err
		}

		// Cache the config and path, then return
		cached_config = config
		cached_config_path = config_path
		return config, nil

	case "global":
		grimoire_home, err := utils.GrimoireHome()
		if err != nil {
			return nil, err
		}
		config_path := grimoire_home + "/grimoire.yaml"

		config, err := config.ParseConfig(config_path)
		if err != nil {
			return nil, err
		}

		// Cache the config and path, then return
		cached_config = config
		cached_config_path = config_path
		return config, nil

	default:
		return nil, fmt.Errorf("invalid config type: %s", config_type)
	}
}
