package scroll

import (
	"os"
	"fmt"
	"strconv"
	"path/filepath"

	"github.com/goccy/go-yaml"

	utils "github.com/jlkendrick/grimoire/internal/utils"
)


// User-facing structs. Simplified version of internal-only FunctionDescriptor.
type Scroll struct {
	RegisteredScrolls []ScrollPath `yaml:"registered_scrolls,omitempty"` // Global grimoire only
	Spells            []Spell  `yaml:"spells,omitempty"` // Repo-level spells
	
	Context ContextType `yaml:"-"`
	Path string `yaml:"-"`
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

type Spell struct {
	Command 	 	string `yaml:"command"`
	Path 	   		string `yaml:"path"`
	Function 		string `yaml:"function"`
	Params  		[]Param  `yaml:"params,omitempty"`
	Interpreter string `yaml:"interpreter,omitempty"`

	ScrollPath 	string `yaml:"-"`
	AbsPath 		string `yaml:"-"`
}

func (s Spell) String() string {
	return fmt.Sprintf("Spell{\n\tCommand: %s,\n\tPath: %s,\n\tFunction: %s,\n\tParams: %v,\n\tInterpreter: %s\n}", s.Command, s.Path, s.Function, s.Params, s.Interpreter)
}

func (s Spell) GenerateYAML() string {
	return fmt.Sprintf("  - name: %s\n    file: %s\n    function: %s\n    args: %v\n    interpreter: %s\n", s.Command, s.Path, s.Function, s.Params, s.Interpreter)
}

func (s Spell) LoadSourceCode() ([]byte, error) {
	p, err := utils.ExpandUserPath(s.Path)
	if err != nil {
		return nil, fmt.Errorf("error resolving path: %w", err)
	}
	source_code, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("error loading source code: %v", err)
	}
	return source_code, nil
}

type Param struct {
	Name 		string `yaml:"name"`
	Type 		string `yaml:"type"`
	Default any 	 `yaml:"default,omitempty"`
}

// UnmarshalYAML normalizes the concrete type of Default when YAML has already
// parsed it into a scalar (e.g. uint8(3) vs int(3)). We intentionally do not
// coerce string defaults like "1" into numeric types here, because user config
// may represent defaults as strings prior to typed casting/validation.
func (p *Param) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type rawParam Param
	var tmp rawParam
	if err := unmarshal(&tmp); err != nil {
		return err
	}

	*p = Param(tmp)

	// Normalize only when YAML produced a non-string scalar.
	switch p.Type {
	case "int":
		switch v := p.Default.(type) {
		case int:
			// ok
		case int8:
			p.Default = int(v)
		case int16:
			p.Default = int(v)
		case int32:
			p.Default = int(v)
		case int64:
			p.Default = int(v)
		case uint:
			p.Default = int(v)
		case uint8:
			p.Default = int(v)
		case uint16:
			p.Default = int(v)
		case uint32:
			p.Default = int(v)
		case uint64:
			p.Default = int(v)
		}
	case "float":
		switch v := p.Default.(type) {
		case float64:
			// ok
		case float32:
			p.Default = float64(v)
		case int:
			p.Default = float64(v)
		case int8:
			p.Default = float64(v)
		case int16:
			p.Default = float64(v)
		case int32:
			p.Default = float64(v)
		case int64:
			p.Default = float64(v)
		case uint:
			p.Default = float64(v)
		case uint8:
			p.Default = float64(v)
		case uint16:
			p.Default = float64(v)
		case uint32:
			p.Default = float64(v)
		case uint64:
			p.Default = float64(v)
		}
	}

	return nil
}

func (p Param) String() string {
	return fmt.Sprintf("Param{\n\t\tName: %s,\n\t\tType: %s,\n\t\tDefault: %v\n\t}", p.Name, p.Type, p.Default)
}

func (p *Param) CastAndSetDefault() error {
	// Cast the default values to the appropriate type
	switch p.Type {

	case "string", "str":
		p.Default = p.Default.(string)

	case "int":
		int_default, err := strconv.Atoi(p.Default.(string))
		if err != nil {
			return fmt.Errorf("error converting default value to int: %v", err)
		}
		p.Default = int_default

	case "bool":
		bool_default, err := strconv.ParseBool(p.Default.(string))
		if err != nil {
			return fmt.Errorf("error converting default value to bool: %v", err)
		}
		p.Default = bool_default

	case "float":
		float_default, err := strconv.ParseFloat(p.Default.(string), 64)
		if err != nil {
			return fmt.Errorf("error converting default value to float: %v", err)
		}
		p.Default = float_default

	default:
		return fmt.Errorf("unsupported type: %s", p.Type)
	}
	
	return nil
}

type ContextType string
const (
	ContextTypeLocal  ContextType = "local"
	ContextTypeGlobal ContextType = "global"
)

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
	cfg.Context = ContextTypeLocal
	return &cfg, nil
}

// RegisterScroll appends scroll_path to the global grimoire's
// registered_projects list and writes the updated config back to disk.
func RegisterScroll(scroll_path string) error {
	cfg, err := LoadScroll("global")
	if err != nil {
		return err
	}
	cfg.RegisteredScrolls = append(cfg.RegisteredScrolls, ScrollPath{Path: scroll_path})
	return cfg.Write()
}
