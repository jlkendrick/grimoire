package runtimes

import (
	"io"
	"fmt"
	"bufio"
	"bytes"
	"os"
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

type RunResult struct {
	Output      []byte
	CacheStatus string
	Runtime     string
}

// Handles the entire execution flow of a function (provision, compile, execute)
func Run(function types.Function, args map[string]interface{}) (*RunResult, error) {
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

	// Print provisioning and casting lines now that both Provision and Compile have run
	// (cache_status for Go is set in Compile, so we wait until here)
	if label, ok := execution_context.StateMap["provision_label"].(string); ok {
		status, _ := execution_context.StateMap["cache_status"].(string)
		fmt.Fprintf(os.Stderr, "◈ %s [····] %s\n", label, status)
	}
	fmt.Fprintf(os.Stderr, "◈ casting spell %s\n\n", function.Name)

	err = adapter.PrepareCommand(&execution_context)
	if err != nil {
		return nil, err
	}

	output, err := Execute(&execution_context)
	if err != nil {
		return nil, err
	}

	result := &RunResult{Output: output}
	if cs, ok := execution_context.StateMap["cache_status"].(string); ok {
		result.CacheStatus = cs
	}
	if rv, ok := execution_context.StateMap["runtime_version"].(string); ok {
		result.Runtime = rv
	}
	return result, nil
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