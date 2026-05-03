package descriptor

import (
	"os"
	"encoding/json"
)

func (d *FunctionDescriptor) Write(path string) error {
	json_content, err := json.Marshal(d)
	if err != nil {
		return err
	}
	return os.WriteFile(path, json_content, 0644)
}

func LoadFunctionDescriptor(path string) (FunctionDescriptor, error) {
	json_content, err := os.ReadFile(path)
	if err != nil {
		return FunctionDescriptor{}, err
	}
	var function_descriptor FunctionDescriptor
	err = json.Unmarshal(json_content, &function_descriptor)
	if err != nil {
		return FunctionDescriptor{}, err
	}
	return function_descriptor, nil
}