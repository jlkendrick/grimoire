package core

import (
	"fmt"
	"bytes"
	"errors"
	"os/exec"
	"strings"
	"encoding/json"

	types "github.com/jlkendrick/sigil/types"
	runtimes "github.com/jlkendrick/sigil/core/runtimes"
)

type RuntimeAdapter interface {
	GenerateCommand(function types.Function) (string, []string, error)
	FormatError(err error) error
}


func ExecuteFunction(function types.Function, args map[string]interface{}) ([]byte, error) {
	// adapter: RuntimeAdapter
	adapter, err := assignAdapter(function)
	if err != nil {
		return nil, err
	}

	json_args, err := json.Marshal(args)
	if err != nil {
		return nil, adapter.FormatError(err)
	}

	binary, flags, err := adapter.GenerateCommand(function)
	if err != nil {
		return nil, adapter.FormatError(err)
	}

	cmd := exec.Command(binary, flags...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	cmd.Stdin = bytes.NewReader(json_args)

	err = cmd.Run()
	if err != nil {
		return nil, adapter.FormatError(errors.New(stderr.String()))
	}

	return stdout.Bytes(), nil
}


func assignAdapter(function types.Function) (RuntimeAdapter, error) {
		if !strings.Contains(function.TargetFile, ".") {
			return nil, fmt.Errorf("no file extension found: %s", function.TargetFile)
		}
	
		file_extensions := strings.Split(function.TargetFile, ".")
		file_extension := file_extensions[len(file_extensions)-1]
		
		switch file_extension {
		case "py":
			return &runtimes.PythonAdapter{}, nil
		default:
		return nil, fmt.Errorf("unsupported file extension: %s", file_extension)
	}
}