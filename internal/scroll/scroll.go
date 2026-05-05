package scroll

import (
	"os"
	"fmt"
	"path/filepath"

	"github.com/goccy/go-yaml"
)


// User-facing structs. Simplified version of internal-only FunctionDescriptor.
type Scroll struct {
	RegisteredScrolls []ScrollPath `yaml:"registered_scrolls,omitempty"` // Global grimoire only
	Spells            []Spell  `yaml:"spells,omitempty"` // Repo-level spells

	Path string `yaml:"-"`
}

func (s *Scroll) String() string {
	return fmt.Sprintf("Scroll(Path: %s, Spells: %v)", s.Path, s.Spells)
}

func (s *Scroll) Write() error {
	yaml_content, err := yaml.MarshalWithOptions(s,
		yaml.Indent(2),
		yaml.IndentSequence(true),
	)
	if err != nil {
		return err
	}
	err = os.WriteFile(s.Path, yaml_content, 0644)
	if err != nil {
		return err
	}
	return nil
}

type ScrollPath struct {
	Path string `yaml:"path"`
}

// InitScroll writes a fresh scroll.yaml in dir and returns the corresponding
// in-memory Config. Returns ErrScrollExists if one already exists at that path.
func InitScroll(dir string, include_boilerplate bool) (*Scroll, error) {
	path := filepath.Join(dir, "scroll.yaml")
	if _, err := os.Stat(path); err == nil {
		return nil, ErrScrollExists
	}

	cfg := Scroll{}
	opts := []yaml.EncodeOption{yaml.Indent(2), yaml.IndentSequence(true)}

	if include_boilerplate {
		cfg.Spells = []Spell{
			{
				Command:  "hello",
				Path:     "path/to/hello_world.py",
				Function: "hello_world",
				Params:   []Param{
					{Name: "n", Type: "int", Default: "1"},
				},
			},
		}
		opts = append(opts, yaml.WithComment(yaml.CommentMap{
			"$.functions[0].name":            []*yaml.Comment{yaml.LineComment("CLI command associated with running the function")},
			"$.functions[0].path":            []*yaml.Comment{yaml.LineComment("Path to the file containing the function")},
			"$.functions[0].function":        []*yaml.Comment{yaml.LineComment("Name of the function to run")},
			"$.functions[0].args[0].name":    []*yaml.Comment{yaml.LineComment("Name of the argument")},
			"$.functions[0].args[0].type":    []*yaml.Comment{yaml.LineComment("Type of the argument")},
			"$.functions[0].args[0].default": []*yaml.Comment{yaml.LineComment("Default value of the argument (optional)")},
		}))
	}

	out, err := yaml.MarshalWithOptions(&cfg, opts...)
	if err != nil {
		return nil, fmt.Errorf("marshaling scroll.yaml: %w", err)
	}
	if err := os.WriteFile(path, out, 0644); err != nil {
		return nil, fmt.Errorf("writing scroll.yaml: %w", err)
	}

	cfg.Path = path
	return &cfg, nil
}

// RegisterScroll appends scroll_path to the global grimoire's
// registered_scrolls list and writes the updated config back to disk.
func RegisterScroll(scroll_path string) error {
	cfg, err := LoadRegistry()
	if err != nil {
		return err
	}
	cfg.RegisteredScrolls = append(cfg.RegisteredScrolls, ScrollPath{Path: scroll_path})
	return cfg.Write()
}
