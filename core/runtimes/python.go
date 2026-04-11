package core

import (
	"os"
	"fmt"
	"strings"
	"path/filepath"

	types "github.com/jlkendrick/sigil/types"
)

type PythonAdapter struct {}

func (a *PythonAdapter) GenerateCommand(function types.Function) (string, []string, error) {
	target_dir := filepath.Dir(function.TargetFile)
	parts := strings.Split(function.TargetFile, "/")
	module := strings.TrimSuffix(parts[len(parts)-1], ".py")

    
  inlineScript := fmt.Sprintf(`
import sys, json, importlib, os

target_dir = os.path.expanduser('%s')
sys.path.append(target_dir)

mod = importlib.import_module('%s')

kwargs = json.loads(sys.stdin.read())
result = getattr(mod, '%s')(**kwargs)

if result is not None:
    if isinstance(result, (dict, list)):
        print(json.dumps(result))
    else:
        print(result)
`, target_dir, module, function.TargetFunction)

  // Return the binary and the flags to execute the string
	p, err := a.GetInterpreter(function)
	if err != nil {
		return "", nil, err
	}
  return p, []string{"-c", inlineScript}, nil
}

func (a *PythonAdapter) FormatError(err error) error {
	return fmt.Errorf("python runtime error: %v", err)
}

func (a *PythonAdapter) GetInterpreter(function types.Function) (string, error) {

	// Option 1: Use the interpreter specified in the YAML
	if function.Interpreter != "" {
		p, err := types.ExpandUserPath(function.Interpreter)
		if err != nil {
			return "", err
		}
		return p, nil
	}

	// Option 2: Search for virtual environment (and requirements.txt for next option)
	expanded_target_file, err := types.ExpandUserPath(function.TargetFile)
	if err != nil {
		return "", err
	}
	start_dir := filepath.Dir(expanded_target_file)
	venvPath, pyProjectPath, requirementsPath, err := findProjectRoot(start_dir)
	if err != nil {
		return "", err
	}
	if venvPath != "" {
		return filepath.Join(venvPath, "bin", "python"), nil
	}
	
	// Option 3: Build new virtual environment from pyproject.toml or requirements.txt [TODO]
	if pyProjectPath != "" {
		return "", nil
	}

	if requirementsPath != "" {
		return "", nil
	}

	return "", fmt.Errorf("interpreter not found")
}

func findProjectRoot(start_dir string) (string, string, string, error) {

	var venvPath string
	var pyProjectPath string
	var requirementsPath string

	check_venvPath := filepath.Join(start_dir, ".venv")
	_, err := os.Stat(check_venvPath)
	if _, err := os.Stat(check_venvPath); err == nil {
		venvPath = check_venvPath
	}

	check_pyProjectPath := filepath.Join(start_dir, "pyproject.toml")
	if _, err := os.Stat(check_pyProjectPath); err == nil {
		pyProjectPath = check_pyProjectPath
	}

	check_requirementsPath := filepath.Join(start_dir, "requirements.txt")
	if _, err := os.Stat(check_requirementsPath); err == nil {
		requirementsPath = check_requirementsPath
	}

	if venvPath != "" || pyProjectPath != "" || requirementsPath != "" {
		return venvPath, pyProjectPath, requirementsPath, nil
	}

	parent_dir := filepath.Dir(start_dir)
	if parent_dir == start_dir {
		return "", "", "", fmt.Errorf("project root not found")
	}

	// Recursively search the parent directory
	new_venvPath, new_pyProjectPath, new_requirementsPath, err := findProjectRoot(parent_dir)
	if err != nil {
		return "", "", "", err
	}
	return new_venvPath, new_pyProjectPath, new_requirementsPath, nil
}