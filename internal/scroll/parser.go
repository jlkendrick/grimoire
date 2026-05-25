package scroll

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/goccy/go-yaml"
)

var validScrollNameRe = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

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

	hasContent := len(config.Spells) > 0 || len(config.Rituals) > 0
	if hasContent {
		if err := resolveAndValidateName(&config, yamlStr); err != nil {
			return nil, err
		}
		if err := validateUniqueCommands(&config); err != nil {
			return nil, err
		}
	}

	return &config, nil
}

// resolveAndValidateName populates config.Name from the directory basename
// when the YAML omits the field, and rejects explicit-empty or malformed
// names when the field is present.
func resolveAndValidateName(config *Scroll, yamlBytes []byte) error {
	if config.Name == "" {
		if hasExplicitTopLevelKey(yamlBytes, "name") {
			return fmt.Errorf("scroll %s: explicit empty `name:` is not allowed; remove the field to use the default or set a value", config.Path)
		}
		config.Name = filepath.Base(filepath.Dir(config.Path))
		return nil
	}
	if !validScrollNameRe.MatchString(config.Name) {
		return fmt.Errorf("scroll %s: invalid name %q (must match [A-Za-z0-9_-]+; dots and whitespace are not allowed)", config.Path, config.Name)
	}
	return nil
}

// validateUniqueCommands enforces that no command appears more than once in
// the same scroll, whether as a spell or a ritual.
func validateUniqueCommands(config *Scroll) error {
	seen := make(map[string]string, len(config.Spells)+len(config.Rituals))
	for _, spell := range config.Spells {
		if existing, ok := seen[spell.Command]; ok {
			return fmt.Errorf("scroll %s: command %q declared more than once (already a %s, redeclared as spell)", config.Path, spell.Command, existing)
		}
		seen[spell.Command] = "spell"
	}
	for _, ritual := range config.Rituals {
		if existing, ok := seen[ritual.Command]; ok {
			return fmt.Errorf("scroll %s: command %q declared more than once (already a %s, redeclared as ritual)", config.Path, ritual.Command, existing)
		}
		seen[ritual.Command] = "ritual"
	}
	return nil
}

// hasExplicitTopLevelKey scans raw YAML for a top-level (column-0) key whose
// name matches `key:`. Used to distinguish an absent field from one written
// as `name:` with no value.
func hasExplicitTopLevelKey(yamlBytes []byte, key string) bool {
	prefix := []byte(key + ":")
	for _, line := range bytes.Split(yamlBytes, []byte("\n")) {
		if i := bytes.IndexByte(line, '#'); i >= 0 {
			line = line[:i]
		}
		if len(line) == 0 || line[0] == ' ' || line[0] == '\t' || line[0] == '-' {
			continue
		}
		if bytes.HasPrefix(line, prefix) {
			return true
		}
	}
	return false
}
