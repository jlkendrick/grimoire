package runtimes

import (
	"os"
	"fmt"
	"bufio"
	"strings"
	"os/exec"
	"path/filepath"

	types "github.com/jlkendrick/grimoire/types"
	utils "github.com/jlkendrick/grimoire/utils"
)

type GoAdapter struct {}

func (a *GoAdapter) Provision(function types.Function) (string, error) {
	// Get the go.mod file hash
	expanded_target_file, err := utils.ExpandUserPath(function.TargetFile)
	if err != nil {
		return "", err
	}

	absolute_start_dir := filepath.Join(filepath.Dir(function.SpellPath), filepath.Dir(expanded_target_file))
	matched_targets, found := utils.UpwardsTraversalForTargets(absolute_start_dir, []string{"go.mod"})
	if !found {
		return "", fmt.Errorf("go.mod not found")
	}

	user_go_mod_path, ok := matched_targets["go.mod"]
	if !ok {
		return "", fmt.Errorf("go.mod not found")
	}

	file_hash, _, err := utils.HashFilePathAndContent(user_go_mod_path)
	if err != nil {
		return "", err
	}

	// Check to see if we already have an env for this hash
	temp_grimoire_dir, err := utils.ExpandUserPath("~/Code/Projects/grimoire/.grimoire")
	if err != nil {
		return "", err
	}
	env_path := filepath.Join(temp_grimoire_dir, "envs", file_hash)
	if _, err := os.Stat(env_path); os.IsNotExist(err) {
		// Create the env
		err = os.MkdirAll(env_path, 0755)
		if err != nil {
			return "", err
		}

		// Run go mod init in the env
		cmd := exec.Command("go", "mod", "init", "grimoire_wrapper")
		cmd.Dir = env_path
		err = cmd.Run()
		if err != nil {
			return "", fmt.Errorf("error running go mod init: %w", err)
		}
	}

	// Write the replace directive to the go.mod file

	// Get the user's module name from the go.mod file
	user_module_name, err := getModuleName(user_go_mod_path)
	if err != nil {
		return "", err
	}

	generated_go_mod_path := filepath.Join(env_path, "go.mod")
	generated_go_mod_content_bytes, err := os.ReadFile(generated_go_mod_path)
	if err != nil {
		return "", fmt.Errorf("error reading generated go.mod: %w", err)
	}
	generated_go_mod_content := string(generated_go_mod_content_bytes)

	// Append the line to require the user's module, if it doesn't already exist
	require_line := "require " + user_module_name + " v0.0.0"
	if !strings.Contains(generated_go_mod_content, require_line) {
		generated_go_mod_content += "\n" + require_line
	}

	// Append the replace directive to the go.mod file if it doesn't already exist
	replace_line := "replace " + user_module_name + " => " + filepath.Dir(user_go_mod_path)
	if !strings.Contains(generated_go_mod_content, replace_line) {
		generated_go_mod_content += "\n" + replace_line
	}

	err = os.WriteFile(generated_go_mod_path, []byte(generated_go_mod_content), 0644)
	if err != nil {
		return "", fmt.Errorf("error writing generated go.mod: %w", err)
	}

	// Do not run `go mod tidy` here: it removes "unused" require lines, and the wrapper
	// module does not yet import the replaced module, so tidy would drop the require
	// we just added even though it is needed for replace to apply on build.

	return env_path, nil
}

func (a *GoAdapter) Compile(function types.Function, interpreter string) error {
	return nil
}

func (a *GoAdapter) PrepareCommand(function types.Function, interpreter string, args map[string]interface{}) (string, []string, []byte, error) {
	return "", nil, nil, nil
}

func (a *GoAdapter) FormatError(err error) error {
	return fmt.Errorf("error executing go function: %w", err)
}

// getModuleName opens a go.mod file and extracts the module path.
// e.g., returns "github.com/james/myproject"
func getModuleName(goModPath string) (string, error) {
	file, err := os.Open(goModPath)
	if err != nil {
		return "", fmt.Errorf("could not open go.mod: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		// Skip empty lines or comments
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		// The first actual directive should be the module declaration
		if strings.HasPrefix(line, "module ") {
			// Extract everything after "module "
			moduleName := strings.TrimSpace(strings.TrimPrefix(line, "module "))
			return moduleName, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading go.mod: %w", err)
	}

	return "", fmt.Errorf("no module declaration found in %s", goModPath)
}