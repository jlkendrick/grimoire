package runtimes

import (
	"os"
	"fmt"
	"strings"
	"os/exec"
	"path/filepath"

	types "github.com/jlkendrick/sigil/types"
	utils "github.com/jlkendrick/sigil/utils"
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
		p, err := utils.ExpandUserPath(function.Interpreter)
		if err != nil {
			return "", err
		}
		return p, nil
	}

	// Option 2: Search for virtual environment (and requirements.txt for next option)
	expanded_target_file, err := utils.ExpandUserPath(function.TargetFile)
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
	
	// Option 3: Build new virtual environment from pyproject.toml or requirements.txt
	if pyProjectPath != "" {
		interpreter, err := buildNewEnvironment(pyProjectPath, "pyproject.toml")
		if err != nil {
			return "", err
		}
		return interpreter, nil
	} else if requirementsPath != "" {
		interpreter, err := buildNewEnvironment(requirementsPath, "requirements.txt")
		if err != nil {
			return "", err
		}
		return interpreter, nil
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

func buildNewEnvironment(dependency_file string, dependency_type string) (string, error) {

	run_venv_cmd := func(venv_path string) error {
		create_cmd := exec.Command("python", "-m", "venv", venv_path)
		err := create_cmd.Run()
		if err != nil {
			return fmt.Errorf("error creating venv: %v", err)
		}
		return nil
	}

	// Create the venv (if it doesn't exist)
	file_hash, content_hash, err := utils.HashFilePathAndContent(dependency_file)
	if err != nil {
		return "", err
	}
	venv_path := filepath.Join(".sigil", "envs", file_hash)
	if _, err := os.Stat(venv_path); os.IsNotExist(err) {
		if err := run_venv_cmd(venv_path); err != nil {
			return "", err
		}
	} else {
		// Check the content hash to see if the venv needs to be updated
		// Content hash is in venv_path/.sigil_req_hash
		content_hash_file := filepath.Join(venv_path, ".sigil_req_hash")
		if _, err := os.Stat(content_hash_file); os.IsNotExist(err) {
			return "", fmt.Errorf("content hash file not found: %v", err)
		}
		content_hash_file_content, err := os.ReadFile(content_hash_file)
		if err != nil {
			return "", fmt.Errorf("error reading content hash file: %v", err)
		}

		// If the content hash is different, create a new venv
		if string(content_hash_file_content) != content_hash {
			if err := run_venv_cmd(venv_path); err != nil {
				return "", err
			}
		}
		// Otherwise, reuse the existing venv
	}

	// venv_path is now the path to the venv

	// Install the dependencies into the venv
	var install_cmd *exec.Cmd
	project_root := filepath.Dir(dependency_file)
	switch dependency_type {
	case "pyproject.toml":
		install_cmd = exec.Command(filepath.Join(venv_path, "bin", "pip"), "install", ".")
		install_cmd.Dir = project_root
	case "requirements.txt":
		install_cmd = exec.Command(filepath.Join(venv_path, "bin", "pip"), "install", "-r", dependency_file)
	default:
		return "", fmt.Errorf("unsupported dependency type: %s", dependency_type)
	}
	if err := install_cmd.Run(); err != nil {
		os.RemoveAll(venv_path)
		return "", fmt.Errorf("error installing dependencies: %v", err)
	}

	// If we succeeded, write the content hash to the venv_path/.sigil_req_hash file. This is our certificate of success.
	content_hash_file := filepath.Join(venv_path, ".sigil_req_hash")
	if err := os.WriteFile(content_hash_file, []byte(content_hash), 0644); err != nil {
		os.RemoveAll(venv_path)
		return "", fmt.Errorf("error writing content hash file: %v", err)
	}

	return filepath.Join(venv_path, "bin", "python"), nil
}