package runtimes

import (
	"os"
	"fmt"
	"strings"
	"os/exec"
	"path/filepath"
	"encoding/json"

	"github.com/pelletier/go-toml/v2"

	types "github.com/jlkendrick/grimoire/types"
	utils "github.com/jlkendrick/grimoire/utils"
)

type PythonAdapter struct {}

func (a *PythonAdapter) Provision(execution_context *ExecutionContext) error {
	function := execution_context.StateMap["function"].(types.Function)

	// Option 1: Use the interpreter specified in the YAML
	if function.Interpreter != "" {
		p, err := utils.ExpandUserPath(function.Interpreter)
		if err != nil {
			return err
		}
		execution_context.StateMap["interpreter"] = p
		return nil
	}

	// Option 2: Search for virtual environment (and requirements.txt for next option)
	expanded_target_file, err := utils.ExpandUserPath(function.TargetFile)
	if err != nil {
		return err
	}
	start_dir := filepath.Dir(expanded_target_file)
	matched_targets, found := utils.UpwardsTraversalForTargets(start_dir, []string{".venv", "pyproject.toml", "requirements.txt"})
	// Option 5: No project root found, use the system interpreter
	if !found {
		execution_context.StateMap["interpreter"] = "python"
		return nil
	}

	// Unpack the matched targets
	var venvPath, pyProjectPath, requirementsPath string
	if venv_path, ok := matched_targets[".venv"]; ok {
		venvPath = venv_path
	}
	if pyproject_path, ok := matched_targets["pyproject.toml"]; ok {
		pyProjectPath = pyproject_path
	}
	if requirements_path, ok := matched_targets["requirements.txt"]; ok {
		requirementsPath = requirements_path
	}

	// Option 3: Use the virtual environment
	if venvPath != "" {
		execution_context.StateMap["interpreter"] = filepath.Join(venvPath, "bin", "python")
		return nil
	}
	
	// Option 4: Build new virtual environment from pyproject.toml or requirements.txt
	abs_function_path := filepath.Join(filepath.Dir(function.SpellPath), function.TargetFile)
	if pyProjectPath != "" {
		interpreter, err := buildNewEnvironment(pyProjectPath, "pyproject.toml", abs_function_path)
		if err != nil {
			return err
		}
		execution_context.StateMap["interpreter"] = interpreter
		return nil
	} else if requirementsPath != "" {
		interpreter, err := buildNewEnvironment(requirementsPath, "requirements.txt", abs_function_path)
		if err != nil {
			return err
		}
		execution_context.StateMap["interpreter"] = interpreter
		return nil
	}

	return fmt.Errorf("should not happen")
}

func (a *PythonAdapter) Compile(execution_context *ExecutionContext) error {
	return nil
}

func (a *PythonAdapter) PrepareCommand(execution_context *ExecutionContext) error {
	function := execution_context.StateMap["function"].(types.Function)
	interpreter := execution_context.StateMap["interpreter"].(string)
	args := execution_context.StateMap["args"].(map[string]interface{})

	target_dir := filepath.Dir(function.TargetFile)
	parts := strings.Split(function.TargetFile, "/")
	module := strings.TrimSuffix(parts[len(parts)-1], ".py")

    
  inlineScript := fmt.Sprintf(`
import sys, json, importlib, os
from contextlib import redirect_stdout

target_dir = os.path.expanduser('%s')
sys.path.append(target_dir)

mod = importlib.import_module('%s')

kwargs = json.loads(sys.stdin.read())
with redirect_stdout(sys.stderr):
    result = getattr(mod, '%s')(**kwargs)

if result is not None:
    if isinstance(result, (dict, list)):
        print(json.dumps(result))
    else:
        print(result)
`, target_dir, module, function.TargetFunction)

  json_args, err := json.Marshal(args)
	if err != nil {
		return err
	}
	
  // Return the binary and the flags to execute the string
  execution_context.StateMap["binary"] = interpreter
  execution_context.StateMap["flags"] = []string{"-c", inlineScript}
  execution_context.StateMap["json_args"] = json_args
  return nil
}

func (a *PythonAdapter) GenerateCommand(function types.Function) (string, []string, error) {
	target_dir := filepath.Dir(function.TargetFile)
	parts := strings.Split(function.TargetFile, "/")
	module := strings.TrimSuffix(parts[len(parts)-1], ".py")

    
  inlineScript := fmt.Sprintf(`
import sys, json, importlib, os
from contextlib import redirect_stdout

target_dir = os.path.expanduser('%s')
sys.path.append(target_dir)

mod = importlib.import_module('%s')

kwargs = json.loads(sys.stdin.read())
with redirect_stdout(sys.stderr):
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
	matched_targets, found := utils.UpwardsTraversalForTargets(start_dir, []string{".venv", "pyproject.toml", "requirements.txt"})
	// Option 5: No project root found, use the system interpreter
	if !found {
		return "python", nil
	}

	// Unpack the matched targets
	var venvPath, pyProjectPath, requirementsPath string
	if venv_path, ok := matched_targets[".venv"]; ok {
		venvPath = venv_path
	}
	if pyproject_path, ok := matched_targets["pyproject.toml"]; ok {
		pyProjectPath = pyproject_path
	}
	if requirements_path, ok := matched_targets["requirements.txt"]; ok {
		requirementsPath = requirements_path
	}

	// Option 3: Use the virtual environment
	if venvPath != "" {
		return filepath.Join(venvPath, "bin", "python"), nil
	}
	
	// Option 4: Build new virtual environment from pyproject.toml or requirements.txt
	abs_function_path := filepath.Join(filepath.Dir(function.SpellPath), function.TargetFile)
	if pyProjectPath != "" {
		interpreter, err := buildNewEnvironment(pyProjectPath, "pyproject.toml", abs_function_path)
		if err != nil {
			return "", err
		}
		return interpreter, nil
	} else if requirementsPath != "" {
		interpreter, err := buildNewEnvironment(requirementsPath, "requirements.txt", abs_function_path)
		if err != nil {
			return "", err
		}
		return interpreter, nil
	}

	return "should not happen", fmt.Errorf("should not happen")
}

type PyProject struct {
	Project struct {
		Dependencies []string `toml:"dependencies"`
	} `toml:"project"`
}

func buildNewEnvironment(dependency_file string, dependency_type string, abs_function_path string) (string, error) {
	run_venv_cmd := func(venv_path string) error {
		create_cmd := exec.Command("python", "-m", "venv", venv_path)
		err := create_cmd.Run()
		if err != nil {
			return fmt.Errorf("error creating venv: %v", err)
		}
		return nil
	}

	// Hash the dependency file and the content of the file
	file_hash, content_hash, err := utils.HashFilePathAndContent(dependency_file)
	if err != nil {
		return "", err
	}
	// For development, put the .grimoire dir in the our local grimoire repo for easy access, will change later to a more permanent location
	temp_grimoire_dir, err := utils.ExpandUserPath("~/Code/Projects/grimoire/.grimoire")
	if err != nil {
		return "", err
	}
	venv_path := filepath.Join(temp_grimoire_dir, "envs", file_hash)

	// If the venv doesn't exist, create it
	if _, err := os.Stat(venv_path); os.IsNotExist(err) {
		if err := run_venv_cmd(venv_path); err != nil {
			return "", err
		}
	} else {
		// Check the content hash to see if the venv needs to be updated
		// Content hash is in venv_path/.grimoire_req_hash
		content_hash_file := filepath.Join(venv_path, ".grimoire_req_hash")
		if _, err := os.Stat(content_hash_file); os.IsNotExist(err) {
			// Hash file missing, treat as stale
			os.RemoveAll(venv_path)
			if err := run_venv_cmd(venv_path); err != nil {
					return "", err
			}
		} else {
			// Hash file exists, check if it's stale
			content_hash_file_content, err := os.ReadFile(content_hash_file)
			if err != nil {
					return "", fmt.Errorf("error reading content hash file: %v", err)
			}
			if string(content_hash_file_content) != content_hash {
					os.RemoveAll(venv_path)
					if err := run_venv_cmd(venv_path); err != nil {
							return "", err
					}
			} else {
				// Hash matches, venv is up-to-date, skip reinstall
				return filepath.Join(venv_path, "bin", "python"), nil
			}
		}
	}

	// venv_path is now the path to the venv

	// Install the dependencies into the venv
	var install_cmd *exec.Cmd
	switch dependency_type {
	case "pyproject.toml":
		file_bytes, err := os.ReadFile(dependency_file)
		if err != nil {
			return "", fmt.Errorf("error reading pyproject.toml: %v", err)
		}

		// Unmarshal the pyproject.toml file into a PyProject struct
		var pyproject PyProject
		if err := toml.Unmarshal(file_bytes, &pyproject); err != nil {
			return "", fmt.Errorf("error unmarshalling pyproject.toml: %v", err)
		}
		dependencies := pyproject.Project.Dependencies
		
		// If there are no dependencies, then skip to final return
		if len(dependencies) > 0 {
			args := append([]string{"install"}, dependencies...)
			install_cmd = exec.Command(filepath.Join(venv_path, "bin", "pip"), args...)
		}

	case "requirements.txt":
		install_cmd = exec.Command(filepath.Join(venv_path, "bin", "pip"), "install", "-r", dependency_file)

	default:
		return "", fmt.Errorf("unsupported dependency type: %s", dependency_type)
	}

	// Install the dependencies into the venv
	if install_cmd != nil {
		if err := install_cmd.Run(); err != nil {
			os.RemoveAll(venv_path)
			return "", fmt.Errorf("error installing dependencies: %v", err)
		}
	}

	// If we succeeded, write the content hash to the venv_path/.grimoire_req_hash file. This is our certificate of success.
	content_hash_file := filepath.Join(venv_path, ".grimoire_req_hash")
	if err := os.WriteFile(content_hash_file, []byte(content_hash), 0644); err != nil {
		os.RemoveAll(venv_path)
		return "", fmt.Errorf("error writing content hash file: %v", err)
	}

	// Write the origin function path to the venv_path/.grimoire_origin file. This is our certificate of origin.
	origin_function_path := filepath.Join(venv_path, ".grimoire_origin")
	if err := os.WriteFile(origin_function_path, []byte(abs_function_path), 0644); err != nil {
		os.RemoveAll(venv_path)
		return "", fmt.Errorf("error writing origin function path file: %v", err)
	}

	return filepath.Join(venv_path, "bin", "python"), nil
}