package core

import (
	"io"
	"fmt"
	"bufio"
	"bytes"
	"os/exec"
	"strings"
	"encoding/json"

	types "github.com/jlkendrick/grimoire/types"
	runtimes "github.com/jlkendrick/grimoire/core/runtimes"
)

type RuntimeAdapter interface {
	GenerateCommand(function types.Function) (string, []string, error)
	FormatError(err error) error
	GetInterpreter(function types.Function) (string, error)
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
		return nil, adapter.FormatError(err)
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