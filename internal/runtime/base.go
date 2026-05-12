package runtime

import (
	"io"
	"os"
	"fmt"
	"sync"
	"bufio"
	"bytes"
	"os/exec"
	"strings"
	"path/filepath"

	descriptor "github.com/jlkendrick/grimoire/internal/descriptor"
	utils "github.com/jlkendrick/grimoire/internal/utils"
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

type RunOptions struct {
	// If set, each line of the spell's stderr is delivered here instead of
	// being printed to os.Stderr by Execute.
	OnStderrLine func(line string)
	// If true, Run does not print its own provisioning / casting-spell lines.
	// The caller is taking responsibility for per-step framing.
	SuppressFraming bool
}

// Handles the entire execution flow of a function (provision, compile, execute)
func Run(descriptor *descriptor.FunctionDescriptor, args map[string]interface{}, opts *RunOptions) (*RunResult, error) {
	if opts == nil {
		opts = &RunOptions{}
	}

	execution_context := ExecutionContext{
		StateMap: make(map[string]any),
	}
	execution_context.StateMap["descriptor"] = descriptor
	execution_context.StateMap["args"] = args
	if opts.OnStderrLine != nil {
		execution_context.StateMap["on_stderr_line"] = opts.OnStderrLine
	}

	// Dynamically assign the appropriate adapter based on the function's target file extension
	adapter, err := assignAdapter(descriptor.AbsPathToSourceFile)
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
	if !opts.SuppressFraming {
		if label, ok := execution_context.StateMap["provision_label"].(string); ok {
			status, _ := execution_context.StateMap["cache_status"].(string)
			fmt.Fprintf(os.Stderr, "%s %s %s %s\n", utils.AccentStyle("◈"), label, utils.AccentStyle("[····]"), utils.DimStyle(status))
		}
		fmt.Fprintf(os.Stderr, "%s casting spell %s\n\n", utils.AccentStyle("◈"), descriptor.CommandName)
	}

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

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	// Start the command but don't wait for it to finish
	cmd.Stdin = bytes.NewReader(json_args)
	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	// Read the stdout and stderr of the command in parallel
	var wg sync.WaitGroup
	
	// Read the stderr of the command. If an OnStderrLine handler is registered
	// in the execution context, route each line there; otherwise print it.
	onStderrLine, _ := execution_context.StateMap["on_stderr_line"].(func(string))
	wg.Add(1)
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			if onStderrLine != nil {
				onStderrLine(line)
			} else {
				fmt.Println(line)
			}
		}
		wg.Done()
	}()
	
	// Read the stdout of the command and store it in a buffer
	var output bytes.Buffer
	wg.Add(1)
	go func() {
		io.Copy(&output, stdout)
		wg.Done()
	}()

	wg.Wait()

	// Wait for the command to finish
	if err = cmd.Wait(); err != nil {
		return nil, err
	}

	return output.Bytes(), nil
}


func assignAdapter(function_path string) (RuntimeAdapter, error) {
		if !strings.Contains(function_path, ".") {
			return nil, fmt.Errorf("no file extension found: %s", function_path)
		}
	
		file_extension := filepath.Ext(function_path)
		
		switch file_extension {
		case ".py":
			return &PythonAdapter{}, nil
		case ".go":
			return &GoAdapter{}, nil
		default:
			return nil, fmt.Errorf("unsupported file extension: %s", file_extension)
	}
}