package runtimes

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"unicode"
	"unicode/utf8"

	types "github.com/jlkendrick/grimoire/types"
	utils "github.com/jlkendrick/grimoire/utils"
)

func uppercaseFirst(s string) string {
	if s == "" {
		return s
	}
	r, size := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError && size == 0 {
		return s
	}
	return string(unicode.ToUpper(r)) + s[size:]
}


type GoAdapter struct {}

func (a *GoAdapter) Provision(execution_context *ExecutionContext) error {
	function := execution_context.StateMap["function"].(types.Function)

	// Get the go.mod file hash
	absolute_start_dir := filepath.Join(filepath.Dir(function.SpellPath), filepath.Dir(function.TargetFile))
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
	grimoire_dir, err := utils.ExpandUserPath("~/.grimoire")
	if err != nil {
		return err
	}
	env_path := filepath.Join(grimoire_dir, "envs", file_hash)
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

	// Get the user's module name from the go.mod file
	user_module_name, err := getModuleName(user_go_mod_path)
	if err != nil {
		return err
	}

	// Read the existing wrapper go.mod to extract the Go toolchain version
	// written by `go mod init` (or a prior run).
	generated_go_mod_path := filepath.Join(env_path, "go.mod")
	generated_go_mod_content_bytes, err := os.ReadFile(generated_go_mod_path)
	if err != nil {
		return fmt.Errorf("error reading generated go.mod: %w", err)
	}

	// Extract the "go X.Y" line from the existing wrapper go.mod.
	go_version_line := "go 1.21"
	for _, line := range strings.Split(string(generated_go_mod_content_bytes), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "go ") {
			go_version_line = trimmed
			break
		}
	}

	// Always regenerate the wrapper go.mod so the replace directive reflects
	// the current user module location. Incremental patching risks duplicate
	// replace directives when the project path changes between runs.
	wrapper_go_mod := fmt.Sprintf(
		"module grimoire_wrapper\n\n%s\n\nrequire %s v0.0.0\n\nreplace %s => %s\n",
		go_version_line, user_module_name, user_module_name, filepath.Dir(user_go_mod_path),
	)
	if err := os.WriteFile(generated_go_mod_path, []byte(wrapper_go_mod), 0644); err != nil {
		return fmt.Errorf("error writing generated go.mod: %w", err)
	}

	execution_context.StateMap["user_go_mod_path"] = user_go_mod_path
	execution_context.StateMap["user_module_name"] = user_module_name
	execution_context.StateMap["env_path"] = env_path
	return nil
}

type WrapperData struct {
	UserModule string
	FuncName   string
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

func shouldCompile(execution_context *ExecutionContext) bool {
	// Check 1: if we haven't compiled before (no binary file exists)
	binary_path := filepath.Join(execution_context.StateMap["env_path"].(string), "grimoire_exec")
	if _, err := os.Stat(binary_path); os.IsNotExist(err) {
		return true
	}

	// Check 2: if any source files have changed since last compile
	// Get compile time of binary
	binary_info, err := os.Stat(binary_path)
	if err != nil {
		return true
	}
	binary_compile_time := binary_info.ModTime()

	// Scan user source files (and go.mod and go.sum) starting from the dir of function's corresponding go.mod file
	user_go_mod_path_dir := filepath.Dir(execution_context.StateMap["user_go_mod_path"].(string))
	needs_recompile := false
	err = filepath.WalkDir(user_go_mod_path_dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if strings.HasPrefix(entry.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "go.mod") || strings.HasSuffix(path, "go.sum") {
			file_info, err := os.Stat(path)
			if err != nil {
				return err
			}
			if file_info.ModTime().After(binary_compile_time) {
				needs_recompile = true
			}
		}
		return nil
	})
	if err != nil {
		return true
	}

	return needs_recompile
}

func (a *GoAdapter) Compile(execution_context *ExecutionContext) error {
	// Check if we need to compile
	if !shouldCompile(execution_context) {
		execution_context.StateMap["binary"] = filepath.Join(execution_context.StateMap["env_path"].(string), "grimoire_exec")
		return nil
	}
	
	function := execution_context.StateMap["function"].(types.Function)
	user_module_name := execution_context.StateMap["user_module_name"].(string)
	user_go_mod_path := execution_context.StateMap["user_go_mod_path"].(string)
	args_def := []ArgDef{}
	for _, arg := range function.Args {
		args_def = append(args_def, ArgDef{
			Name: uppercaseFirst(arg.Name),
			Type: arg.Type,
			Key: strings.ToLower(arg.Name),
		})
	}

	// Calculate the import path for the user's module.
	absolute_function_path := filepath.Join(filepath.Dir(function.SpellPath), function.TargetFile)
	user_module_path, err := utils.MakeRelativePath(filepath.Dir(absolute_function_path), filepath.Dir(user_go_mod_path))
	if err != nil {
		return err
	}
	// When the function lives in the module root, MakeRelativePath returns ".".
	// "module/." is not a valid Go import path; use the module name directly.
	var user_import_path string
	if user_module_path == "." {
		user_import_path = user_module_name
	} else {
		user_import_path = user_module_name + "/" + user_module_path
	}
	wrapper_data := WrapperData{
		UserModule: user_import_path,
		FuncName:   function.TargetFunction,
		Args:       args_def,
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
	cmd = exec.Command("go", "build", "-o", "grimoire_exec", ".")
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