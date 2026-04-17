package runtimes

import (
	"fmt"

	types "github.com/jlkendrick/grimoire/types"
)

type GoAdapter struct {}

func (a *GoAdapter) Provision(function types.Function) (string, error) {
	return "go", nil
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