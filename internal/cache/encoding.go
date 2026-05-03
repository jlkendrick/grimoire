package cache

import (
	"os"
	"path/filepath"
	"encoding/json"

	utils "github.com/jlkendrick/grimoire/internal/utils"
	descriptor "github.com/jlkendrick/grimoire/internal/descriptor"
)

func cachePath() (string, error) {
	grimoire_home, err := utils.GrimoireHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(grimoire_home, "cache.json"), nil
}

func WriteCache(descriptors []descriptor.FunctionDescriptor) error {
	cache_path, err := cachePath()
	if err != nil {
		return err
	}
	cache_file, err := os.Create(cache_path)
	if err != nil {
		return err
	}
	defer cache_file.Close()
	json.NewEncoder(cache_file).Encode(descriptors)
	return nil
}

func LoadCache() ([]descriptor.FunctionDescriptor, error) {
	cache := []descriptor.FunctionDescriptor{}
	cache_path, err := cachePath()
	if err != nil {
		return nil, err
	}
	cache_file, err := os.Open(cache_path)
	if err != nil {
		return nil, err
	}
	defer cache_file.Close()
	json.NewDecoder(cache_file).Decode(&cache)
	return cache, nil
}