package core

import (
	"fmt"
	"strings"

	types "github.com/jlkendrick/sigil/types"
)

type RuntimeAdapter interface {
	GenerateCommand(function types.Function, args []string) (string, error)
	FormatError(err error) error
}

func ExecuteFunction(function types.Function, args []string) error {
	// adapter: RuntimeAdapter
	adapter, err := assignAdapter(function)
	if err != nil {
		return err
	}

	cmd, err := adapter.GenerateCommand(function, args)
	if err != nil {
		return err
	}

	err = cmd.Execute()
	if err != nil {
		return err
	}
	return nil
}

func assignAdapter(function types.Function) (RuntimeAdapter, error) {
		if !strings.Contains(function.TargetFile, ".") {
			return nil, fmt.Errorf("no file extension found: %s", function.TargetFile)
		}
	
		file_extensions := strings.Split(function.TargetFile, ".")
		file_extension := file_extensions[len(file_extensions)-1]
		
		switch file_extension {
		case "py":
			return &PythonAdapter{}, nil
		default:
		return nil, fmt.Errorf("unsupported file extension: %s", file_extension)
	}
}