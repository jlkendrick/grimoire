package cache

import (
	"os"
	"encoding/json"

	descriptor "github.com/jlkendrick/grimoire/internal/descriptor"
)

func WriteCache(descriptors []descriptor.FunctionDescriptor, path string) error {
	cache_file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer cache_file.Close()
	json.NewEncoder(cache_file).Encode(descriptors)
	return nil
}

func LoadCache(path string) ([]descriptor.FunctionDescriptor, error) {
	cache := []descriptor.FunctionDescriptor{}
	cache_file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer cache_file.Close()
	json.NewDecoder(cache_file).Decode(&cache)
	return cache, nil
}