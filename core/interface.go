package core

import (
	"io"
	"fmt"
	"bufio"
	"bytes"
	"os/exec"
	"strings"

	types "github.com/jlkendrick/grimoire/types"
	runtimes "github.com/jlkendrick/grimoire/core/runtimes"
)

type RuntimeAdapter interface {
	Provision(function types.Function) (string, error)
	Compile(function types.Function, interpreter string) error
	PrepareCommand(function types.Function, interpreter string, args map[string]interface{}) (string, []string, []byte, error)

	FormatError(err error) error
}

// Handles the entire execution flow of a function (provision, compile, execute)
func Run(function types.Function, args map[string]interface{}) ([]byte, error) {

	// Dynamically assign the appropriate adapter based on the function's target file extension
	adapter, err := assignAdapter(function)
	if err != nil {
		return nil, err
	}

	// Provision the runtime environment
	interpreter, err := adapter.Provision(function)
	if err != nil {
		return nil, err
	}

	// Compile the function (no-op for non-compiled languages)
	err = adapter.Compile(function, interpreter)
	if err != nil {
		return nil, err
	}

	binary, flags, json_args, err := adapter.PrepareCommand(function, interpreter, args)
	if err != nil {
		return nil, err
	}

	output, err := Execute(binary, flags, json_args)
	if err != nil {
		return nil, err
	}

	return output, nil
}

func Execute(binary string, flags []string, json_args []byte) ([]byte, error) {

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
			return &runtimes.PythonAdapter{}, nil
		default:
		return nil, fmt.Errorf("unsupported file extension: %s", file_extension)
	}
}