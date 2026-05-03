package scroll

import (
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"
)

// Parse the user's scroll.yaml file
func ParseScroll(path string) (*Scroll, error) {
	yamlStr, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Scroll
	if err := yaml.Unmarshal([]byte(yamlStr), &config); err != nil {
		return nil, err
	}

	// If the config is global, parse the individual project configs and set the context type to global
	if config.RegisteredScrolls != nil {
		for _, scroll := range config.RegisteredScrolls {
			scroll_config, err := ParseScroll(scroll.Path)
			if err != nil {
				return nil, err
			}
			// Store a reference to the scroll path that the function originally belongs to
			// Have to do this here so we don't lose what project the function originally belonged to
			for i := range scroll_config.Spells {
				scroll_config.Spells[i].ScrollPath = scroll.Path
				scroll_config.Spells[i].AbsPath = filepath.Join(filepath.Dir(scroll.Path), scroll_config.Spells[i].Path)
			}
			config.Spells = append(config.Spells, scroll_config.Spells...)
		}

		config.Context = ContextTypeGlobal
	
	} else {
		// Need to set the ScrollPaths here as well for clean command to work with local spells
		for i := range config.Spells {
			config.Spells[i].ScrollPath = path
			config.Spells[i].AbsPath = filepath.Join(filepath.Dir(path), config.Spells[i].Path)
		}
		config.Context = ContextTypeLocal
	}

	config.Path = path

	return &config, nil
}