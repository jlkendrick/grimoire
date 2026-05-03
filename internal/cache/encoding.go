package cache

import (
	"os"
	"path/filepath"
	"encoding/json"

	utils "github.com/jlkendrick/grimoire/internal/utils"
	descriptor "github.com/jlkendrick/grimoire/internal/descriptor"
)

var cached_descriptors *[]descriptor.FunctionDescriptor
var cached_cache_path string

func ResetCache() {
	cached_descriptors = nil
	cached_cache_path = ""
}

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

func LoadCache() (*[]descriptor.FunctionDescriptor, error) {
	if cached_descriptors != nil {
		return cached_descriptors, nil
	}

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

	// Cache the descriptors and path, then return
	cached_descriptors = &cache
	cached_cache_path = cache_path
	
	return cached_descriptors, nil
}