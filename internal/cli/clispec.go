package cli

type CLISpec struct {
	Command string
	Flags   []FlagSpec
}

type FlagSpec struct {
	Name        string  // "--host"
	GoType      string  // "string", "int", "bool"
	Required    bool
	Default     interface{}
	Description string
}