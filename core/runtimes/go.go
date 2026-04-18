package runtimes

import (
	"os"
	"fmt"
	"bufio"
	"strings"
	"os/exec"
	"path/filepath"
	"text/template"
	"encoding/json"

	types "github.com/jlkendrick/grimoire/types"
	utils "github.com/jlkendrick/grimoire/utils"
)


type GoAdapter struct {}

func (a *GoAdapter) Provision(execution_context *ExecutionContext) error {
	function := execution_context.StateMap["function"].(types.Function)
	
	// Get the go.mod file hash
	expanded_target_file, err := utils.ExpandUserPath(function.TargetFile)
	if err != nil {
		return err
	}

	absolute_start_dir := filepath.Join(filepath.Dir(function.SpellPath), filepath.Dir(expanded_target_file))
	matched_targets, found := utils.UpwardsTraversalForTargets(absolute_start_dir, []string{"go.mod"})
	if !found {
		return fmt.Errorf("go.mod not found")
	}

	user_go_mod_path, ok := matched_targets["go.mod"]
	if !ok {
		return fmt.Errorf("go.mod not found")
	}

	file_hash, _, err := utils.HashFilePathAndContent(user_go_mod_path)
	if err != nil {
		return err
	}

	// Check to see if we already have an env for this hash
	temp_grimoire_dir, err := utils.ExpandUserPath("~/Code/Projects/grimoire/.grimoire")
	if err != nil {
		return err
	}
	env_path := filepath.Join(temp_grimoire_dir, "envs", file_hash)
	if _, err := os.Stat(env_path); os.IsNotExist(err) {
		// Create the env
		err = os.MkdirAll(env_path, 0755)
		if err != nil {
			return err
		}

		// Run go mod init in the env
		cmd := exec.Command("go", "mod", "init", "grimoire_wrapper")
		cmd.Dir = env_path
		err = cmd.Run()
		if err != nil {
			return fmt.Errorf("error running go mod init: %w", err)
		}
	}

	// Write the replace directive to the go.mod file

	// Get the user's module name from the go.mod file
	user_module_name, err := getModuleName(user_go_mod_path)
	if err != nil {
		return err
	}

	generated_go_mod_path := filepath.Join(env_path, "go.mod")
	generated_go_mod_content_bytes, err := os.ReadFile(generated_go_mod_path)
	if err != nil {
		return fmt.Errorf("error reading generated go.mod: %w", err)
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
		return fmt.Errorf("error writing generated go.mod: %w", err)
	}

	// Do not run `go mod tidy` here: it removes "unused" require lines, and the wrapper
	// module does not yet import the replaced module, so tidy would drop the require
	// we just added even though it is needed for replace to apply on build.

	execution_context.StateMap["user_go_mod_path"] = user_go_mod_path
	execution_context.StateMap["user_module_name"] = user_module_name
	execution_context.StateMap["generated_go_mod_path"] = generated_go_mod_path
	execution_context.StateMap["env_path"] = env_path
	return nil
}

type WrapperData struct {
	UserModule string // e.g., "github.com/james/myproject/api"
	FuncName   string // e.g., "Calculate"
	Args       []ArgDef
}

type ArgDef struct {
	Name string // e.g., "A", "B", "Message" (Title-cased for JSON exporting)
	Type string // e.g., "int", "string", "bool"
	Key  string // e.g., "a", "b", "message" (The lowercase JSON key)
}

const wrapperTemplate = `
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	
	userpkg "{{ .UserModule }}" 
)

// Grimoire-generated struct for strict JSON unmarshaling
type Input struct {
{{- range .Args }}
	{{ .Name }} {{ .Type }} ` + "`json:\"{{ .Key }}\"`" + `
{{- end }}
}

func main() {
	var args Input
	// Read the JSON arguments fed by Grimoire's core engine
	if err := json.NewDecoder(os.Stdin).Decode(&args); err != nil {
		fmt.Fprintf(os.Stderr, "Grimoire Decoder Error: %v\n", err)
		os.Exit(1)
	}

	// Invoke user function and support both returning and non-returning functions.
	fn := reflect.ValueOf(userpkg.{{ .FuncName }})
	results := fn.Call([]reflect.Value{
		{{- range $index, $arg := .Args }}
			{{- if $index }}, {{ end }}reflect.ValueOf(args.{{ $arg.Name }})
		{{- end }}})

	var result interface{}
	switch len(results) {
	case 0:
		result = nil
	case 1:
		result = results[0].Interface()
	default:
		out := make([]interface{}, len(results))
		for i, r := range results {
			out[i] = r.Interface()
		}
		result = out
	}

	// Wrap the result in JSON and send it back to Grimoire
	if err := json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
		"result": result,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Grimoire Encoder Error: %v\n", err)
		os.Exit(1)
	}
}
`

func generateWrapper(outputPath string, data WrapperData) error {
	tmpl, err := template.New("wrapper").Parse(wrapperTemplate)
	if err != nil {
		return err
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	return tmpl.Execute(file, data)
}

func (a *GoAdapter) Compile(execution_context *ExecutionContext) error {
	function := execution_context.StateMap["function"].(types.Function)
	user_module_name := execution_context.StateMap["user_module_name"].(string)
	user_go_mod_path := execution_context.StateMap["user_go_mod_path"].(string)
	args_def := []ArgDef{}
	for _, arg := range function.Args {
		args_def = append(args_def, ArgDef{
			Name: strings.Title(arg.Name),
			Type: arg.Type,
			Key: strings.ToLower(arg.Name),
		})
	}

	// Calculate the import path for the user's module
	absolute_function_path := filepath.Join(filepath.Dir(function.SpellPath), function.TargetFile)
	user_module_path, err := utils.MakeRelativePath(filepath.Dir(absolute_function_path), filepath.Dir(user_go_mod_path))
	if err != nil {
		return err
	}
	wrapper_data := WrapperData{
		UserModule: user_module_name + "/" + user_module_path,
		FuncName: function.TargetFunction,
		Args: args_def,
	}

	output_path := filepath.Join(execution_context.StateMap["env_path"].(string), "grimoire_wrapper.go")
	err = generateWrapper(output_path, wrapper_data)
	if err != nil {
		return err
	}

	// Now run go mod tidy in the env
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = execution_context.StateMap["env_path"].(string)
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("error running go mod tidy: %w", err)
	}

	// Now run go build in the env
	cmd = exec.Command("go", "build", "-o", "grimoire_exec", output_path)
	cmd.Dir = execution_context.StateMap["env_path"].(string)
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("error running go build: %w", err)
	}

	execution_context.StateMap["binary"] = filepath.Join(execution_context.StateMap["env_path"].(string), "grimoire_exec")
	return nil
}

func (a *GoAdapter) PrepareCommand(execution_context *ExecutionContext) error {
	execution_context.StateMap["flags"] = []string{}
	json_args, err := json.Marshal(execution_context.StateMap["args"])
	if err != nil {
		return fmt.Errorf("error marshalling args: %w", err)
	}
	execution_context.StateMap["json_args"] = json_args
	return nil
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