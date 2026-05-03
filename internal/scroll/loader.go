package scroll

import (
	"os"
	"fmt"

	utils "github.com/jlkendrick/grimoire/internal/utils"
)

var cached_scroll *Scroll
var cached_scroll_path string

// ResetConfigCache clears all cached config state. Intended for use in tests.
func ResetScrollCache() {
	cached_scroll = nil
	cached_scroll_path = ""
}

func LoadScroll(scroll_type string) (*Scroll, error) {
	if cached_scroll != nil {
		return cached_scroll, nil
	}

	switch scroll_type {
	case "local":
		current_dir, err := os.Getwd()
		if err != nil {
			return nil, err
		}

		// Determine the path to the config file
		var scroll_path string
		matched_targets, found := utils.UpwardsTraversalForTargets(current_dir, []string{"scroll.yaml"})
		if found {
			scroll_path = matched_targets["scroll.yaml"]
		} else {
			// Fall back to the global grimoire config
			grimoire_home, err := utils.GrimoireHome()
			if err != nil {
				return nil, err
			}
			scroll_path = grimoire_home + "/grimoire.yaml"
		}

		// Parse the config file
		scroll, err := ParseScroll(scroll_path)
		if err != nil {
			return nil, err
		}

		// Cache the config and path, then return
		cached_scroll = scroll
		cached_scroll_path = scroll_path
		return scroll, nil

	case "global":
		grimoire_home, err := utils.GrimoireHome()
		if err != nil {
			return nil, err
		}
		scroll_path := grimoire_home + "/grimoire.yaml"

		scroll, err := ParseScroll(scroll_path)
		if err != nil {
			return nil, err
		}

		// Cache the config and path, then return
		cached_scroll = scroll
		cached_scroll_path = scroll_path
		return scroll, nil

	default:
		return nil, fmt.Errorf("invalid scroll type: %s", scroll_type)
	}
}
