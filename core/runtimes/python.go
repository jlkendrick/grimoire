package core

import (
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
  return "python", []string{"-c", inlineScript}, nil
}

func (a *PythonAdapter) FormatError(err error) error {
	return fmt.Errorf("python runtime error: %v", err)
}