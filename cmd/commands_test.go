package cmd

import (
	"testing"

	types "github.com/jlkendrick/sigil/types"
)

func TestGenerateCommands(t *testing.T) {
	t.Run("empty config", func(t *testing.T) {
		cfg := &types.Config{Functions: []types.Function{}}
		cmds, err := GenerateCommands(cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(cmds) != 0 {
			t.Errorf("expected 0 commands, got %d", len(cmds))
		}
	})

	t.Run("function with no args", func(t *testing.T) {
		cfg := &types.Config{
			Functions: []types.Function{
				{Name: "greet"},
			},
		}
		cmds, err := GenerateCommands(cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(cmds) != 1 {
			t.Fatalf("expected 1 command, got %d", len(cmds))
		}
		if cmds[0].Use != "greet" {
			t.Errorf("expected Use=%q, got %q", "greet", cmds[0].Use)
		}
	})

	t.Run("string arg", func(t *testing.T) {
		cfg := &types.Config{
			Functions: []types.Function{
				{
					Name: "greet",
					Args: []types.Arg{
						{Name: "name", Type: "string", Default: "world"},
					},
				},
			},
		}
		cmds, err := GenerateCommands(cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		flag := cmds[0].Flags().Lookup("name")
		if flag == nil {
			t.Fatal("expected flag 'name' to be registered")
		}
		if flag.DefValue != "world" {
			t.Errorf("expected default %q, got %q", "world", flag.DefValue)
		}
	})

	t.Run("int arg", func(t *testing.T) {
		cfg := &types.Config{
			Functions: []types.Function{
				{
					Name: "count",
					Args: []types.Arg{
						{Name: "n", Type: "int", Default: 5},
					},
				},
			},
		}
		cmds, err := GenerateCommands(cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		flag := cmds[0].Flags().Lookup("n")
		if flag == nil {
			t.Fatal("expected flag 'n' to be registered")
		}
		if flag.DefValue != "5" {
			t.Errorf("expected default %q, got %q", "5", flag.DefValue)
		}
	})

	t.Run("bool arg", func(t *testing.T) {
		cfg := &types.Config{
			Functions: []types.Function{
				{
					Name: "toggle",
					Args: []types.Arg{
						{Name: "verbose", Type: "bool", Default: true},
					},
				},
			},
		}
		cmds, err := GenerateCommands(cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		flag := cmds[0].Flags().Lookup("verbose")
		if flag == nil {
			t.Fatal("expected flag 'verbose' to be registered")
		}
		if flag.DefValue != "true" {
			t.Errorf("expected default %q, got %q", "true", flag.DefValue)
		}
	})

	t.Run("float arg", func(t *testing.T) {
		cfg := &types.Config{
			Functions: []types.Function{
				{
					Name: "scale",
					Args: []types.Arg{
						{Name: "factor", Type: "float", Default: float64(1.5)},
					},
				},
			},
		}
		cmds, err := GenerateCommands(cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		flag := cmds[0].Flags().Lookup("factor")
		if flag == nil {
			t.Fatal("expected flag 'factor' to be registered")
		}
		if flag.DefValue != "1.5" {
			t.Errorf("expected default %q, got %q", "1.5", flag.DefValue)
		}
	})

	t.Run("multiple functions", func(t *testing.T) {
		cfg := &types.Config{
			Functions: []types.Function{
				{Name: "foo"},
				{Name: "bar"},
				{Name: "baz"},
			},
		}
		cmds, err := GenerateCommands(cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(cmds) != 3 {
			t.Fatalf("expected 3 commands, got %d", len(cmds))
		}
		names := map[string]bool{}
		for _, c := range cmds {
			names[c.Use] = true
		}
		for _, want := range []string{"foo", "bar", "baz"} {
			if !names[want] {
				t.Errorf("missing command %q in generated commands", want)
			}
		}
	})

	t.Run("multiple args on one function", func(t *testing.T) {
		cfg := &types.Config{
			Functions: []types.Function{
				{
					Name: "run",
					Args: []types.Arg{
						{Name: "host", Type: "string", Default: "localhost"},
						{Name: "port", Type: "int", Default: 8080},
						{Name: "debug", Type: "bool", Default: false},
					},
				},
			},
		}
		cmds, err := GenerateCommands(cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		for _, flagName := range []string{"host", "port", "debug"} {
			if cmds[0].Flags().Lookup(flagName) == nil {
				t.Errorf("expected flag %q to be registered", flagName)
			}
		}
	})

	// Error cases

	t.Run("unsupported type", func(t *testing.T) {
		cfg := &types.Config{
			Functions: []types.Function{
				{
					Name: "f",
					Args: []types.Arg{
						{Name: "x", Type: "list", Default: nil},
					},
				},
			},
		}
		_, err := GenerateCommands(cfg)
		if err == nil {
			t.Fatal("expected error for unsupported type, got nil")
		}
	})

	t.Run("wrong default type for string", func(t *testing.T) {
		cfg := &types.Config{
			Functions: []types.Function{
				{
					Name: "f",
					Args: []types.Arg{
						{Name: "x", Type: "string", Default: 42},
					},
				},
			},
		}
		_, err := GenerateCommands(cfg)
		if err == nil {
			t.Fatal("expected error for non-string default on string arg, got nil")
		}
	})

	t.Run("wrong default type for int", func(t *testing.T) {
		cfg := &types.Config{
			Functions: []types.Function{
				{
					Name: "f",
					Args: []types.Arg{
						{Name: "x", Type: "int", Default: "not-an-int"},
					},
				},
			},
		}
		_, err := GenerateCommands(cfg)
		if err == nil {
			t.Fatal("expected error for non-int default on int arg, got nil")
		}
	})

	t.Run("wrong default type for bool", func(t *testing.T) {
		cfg := &types.Config{
			Functions: []types.Function{
				{
					Name: "f",
					Args: []types.Arg{
						{Name: "x", Type: "bool", Default: "yes"},
					},
				},
			},
		}
		_, err := GenerateCommands(cfg)
		if err == nil {
			t.Fatal("expected error for non-bool default on bool arg, got nil")
		}
	})

	t.Run("wrong default type for float", func(t *testing.T) {
		cfg := &types.Config{
			Functions: []types.Function{
				{
					Name: "f",
					Args: []types.Arg{
						{Name: "x", Type: "float", Default: "1.5"},
					},
				},
			},
		}
		_, err := GenerateCommands(cfg)
		if err == nil {
			t.Fatal("expected error for non-float64 default on float arg, got nil")
		}
	})

	t.Run("error on second function does not return partial results", func(t *testing.T) {
		cfg := &types.Config{
			Functions: []types.Function{
				{Name: "good"},
				{
					Name: "bad",
					Args: []types.Arg{
						{Name: "x", Type: "unknown", Default: nil},
					},
				},
			},
		}
		_, err := GenerateCommands(cfg)
		if err == nil {
			t.Fatal("expected error when a later function has an unsupported type, got nil")
		}
	})
}
