package core

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	types "github.com/jlkendrick/grimoire/types"
	utils "github.com/jlkendrick/grimoire/utils"

	"github.com/goccy/go-yaml"
)

// ErrScrollExists is returned by InitScroll when a scroll.yaml already exists
// in the target directory.
var ErrScrollExists = errors.New("scroll already exists")

// FindLocalScroll walks upward from start_dir looking for scroll.yaml. Returns
// the absolute path and true on hit, ("", false) otherwise.
func FindLocalScroll(start_dir string) (string, bool) {
	matched, found := utils.UpwardsTraversalForTargets(start_dir, []string{"scroll.yaml"})
	if !found {
		return "", false
	}
	return matched["scroll.yaml"], true
}

// InitScroll writes a fresh scroll.yaml in dir and returns the corresponding
// in-memory Config. Returns ErrScrollExists if one already exists at that path.
func InitScroll(dir string, include_boilerplate bool) (*types.Config, error) {
	path := filepath.Join(dir, "scroll.yaml")
	if _, err := os.Stat(path); err == nil {
		return nil, ErrScrollExists
	}

	cfg := types.Config{}
	opts := []yaml.EncodeOption{yaml.Indent(2), yaml.IndentSequence(true)}

	if include_boilerplate {
		cfg.Functions = []types.Function{
			{
				Name:           "hello_world",
				TargetFile:     "path/to/hello_world.py",
				TargetFunction: "hello_world",
				Args: []types.Arg{
					{Name: "n", Type: "int", Default: 1},
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
	cfg.Context = types.ContextTypeLocal
	return &cfg, nil
}

// RegisterScroll appends scroll_path to the global grimoire's
// registered_projects list and writes the updated config back to disk.
func RegisterScroll(scroll_path string) error {
	cfg, err := LoadConfig("global")
	if err != nil {
		return err
	}
	cfg.RegisteredProjects = append(cfg.RegisteredProjects, types.Project{Path: scroll_path})
	return cfg.Write()
}
