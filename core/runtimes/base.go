package runtimes

import (
	"io"
	"fmt"
	"bufio"
	"bytes"
	"os/exec"
	"strings"

	types "github.com/jlkendrick/grimoire/types"
)

type ExecutionContext struct {
	StateMap map[string]any
}

type RuntimeAdapter interface {
	Provision(execution_context *ExecutionContext) error
	Compile(execution_context *ExecutionContext) error
	PrepareCommand(execution_context *ExecutionContext) error

	FormatError(err error) error
}

// Handles the entire execution flow of a function (provision, compile, execute)
func Run(function types.Function, args map[string]interface{}) ([]byte, error) {
	execution_context := ExecutionContext{
		StateMap: make(map[string]any),
	}
	execution_context.StateMap["function"] = function
	execution_context.StateMap["args"] = args

	// Dynamically assign the appropriate adapter based on the function's target file extension
	adapter, err := assignAdapter(function)
	if err != nil {
		return nil, err
	}

	// Provision the runtime environment
	err = adapter.Provision(&execution_context)
	if err != nil {
		return nil, err
	}

	// Compile the function (no-op for non-compiled languages)
	err = adapter.Compile(&execution_context)
	if err != nil {
		return nil, err
	}

	err = adapter.PrepareCommand(&execution_context)
	if err != nil {
		return nil, err
	}

	output, err := Execute(&execution_context)
	if err != nil {
		return nil, err
	}

	return output, nil
}

func Execute(execution_context *ExecutionContext) ([]byte, error) {
	binary := execution_context.StateMap["binary"].(string)
	flags := execution_context.StateMap["flags"].([]string)
	json_args := execution_context.StateMap["json_args"].([]byte)

	cmd := exec.Command(binary, flags...)

	stderr, _ := cmd.StderrPipe()
	stdout, _ := cmd.StdoutPipe()

	// Start the command but don't wait for it to finish
	cmd.Stdin = bytes.NewReader(json_args)
	cmd.Start()

	// Read the stderr of the command and print it to the console
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}
	}()

	// Read the stdout of the command and store it in a buffer
	var output bytes.Buffer
	io.Copy(&output, stdout)

	// Wait for the command to finish
	if err := cmd.Wait(); err != nil {
		return nil, err
	}

	return output.Bytes(), nil
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
		case "go":
			return &GoAdapter{}, nil
		default:
			return nil, fmt.Errorf("unsupported file extension: %s", file_extension)
	}
}