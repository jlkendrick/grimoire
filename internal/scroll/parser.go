package scroll

import (
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"
)

// ParseScroll deserializes a single scroll YAML file (either a local
// scroll.yaml or the global grimoire.yaml). It does not recurse into
// RegisteredScrolls — see loader.go's LoadScrolls for the multi-scroll flow.
func ParseScroll(path string) (*Scroll, error) {
	yamlStr, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Scroll
	if err := yaml.Unmarshal([]byte(yamlStr), &config); err != nil {
		return nil, err
	}

	for i := range config.Spells {
		config.Spells[i].ScrollPath = path
		config.Spells[i].AbsPath = filepath.Join(filepath.Dir(path), config.Spells[i].Path)
	}

	config.Path = path
	return &config, nil
}
