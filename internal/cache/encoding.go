package cache

import (
	"os"
	"fmt"
	"bytes"
	"path/filepath"
	"encoding/json"

	utils "github.com/jlkendrick/grimoire/internal/utils"
	descriptor "github.com/jlkendrick/grimoire/internal/descriptor"
)

// Cache format:
//
// ~/.grimoire/cache/
//   <scroll-id>.json          # all descriptors for one scroll
//   <scroll-id>.json
//   ...
//
// scroll-id is the hash of the scroll path

const CACHE_VERSION = 1

type DescriptorCache struct {
	Version 	 int 							                        `json:"version"`
	ScrollHash string 							                    `json:"scroll_hash"`
	ScrollPath string 							                    `json:"scroll_path"`
	Functions  map[string]descriptor.FunctionDescriptor `json:"functions"` // function name -> function descriptor
}

var cached_descriptor_cache *DescriptorCache

func ResetCache() {
	cached_descriptor_cache = nil
}

func cachePath() (string, error) {
	grimoire_home, err := utils.GrimoireHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(grimoire_home, "cache"), nil
}

func ReadDescriptorCache(scroll_path string) (*DescriptorCache, error) {
	// If the scroll path is the global grimoire, we need to load all the function descriptors and merge them
	file_name := filepath.Base(scroll_path)
	switch file_name {
	case "grimoire.yaml":
		// Load all the function descriptors and merge them
		cache_path, err := cachePath()
		if err != nil {
			return nil, err
		}
		
		// Iterate over all the files in the cache directory to load all the function descriptors
		files, err := os.ReadDir(cache_path)
		if err != nil {
			return nil, err
		}
		function_descriptors := map[string]descriptor.FunctionDescriptor{}
		for _, file := range files {
			if file.IsDir() {
				continue
			}
			cache_data, err := os.ReadFile(filepath.Join(cache_path, file.Name()))
			if err != nil {
				return nil, err
			}
			var scroll_cache DescriptorCache
			err = json.Unmarshal(cache_data, &scroll_cache)
			if err != nil {
				return nil, err
			}
			for function_name, function_descriptor := range scroll_cache.Functions {
				function_descriptors[function_name] = function_descriptor
			}
		}

		// Create our ephemeral global descriptor cache
		scroll_hash, err := utils.HashFile(scroll_path)
		if err != nil {
			return nil, err
		}
		cached_descriptor_cache = &DescriptorCache{
			Version: CACHE_VERSION,
			ScrollHash: scroll_hash,
			ScrollPath: scroll_path,
			Functions: function_descriptors,
		}
		return cached_descriptor_cache, nil

	case "scroll.yaml":
		if cached_descriptor_cache != nil && cached_descriptor_cache.ScrollPath == scroll_path {
			return cached_descriptor_cache, nil
		}

		cache_path, err := cachePath()
		if err != nil {
			return nil, err
		}
		
		filename_hash, err := utils.HashFilePath(scroll_path)
		if err != nil {
			return nil, err
		}
		cache_file := filepath.Join(cache_path, filename_hash + ".json")
		if _, err := os.Stat(cache_file); os.IsNotExist(err) {
			// If the cache file doesn't exist, create it
			scroll_hash, err := utils.HashFile(scroll_path)
			if err != nil {
				return nil, err
			}
			cached_descriptor_cache = &DescriptorCache{
				Version: CACHE_VERSION,
				ScrollHash: scroll_hash,
				ScrollPath: scroll_path,
				Functions: make(map[string]descriptor.FunctionDescriptor),
			}
			return cached_descriptor_cache, nil
		}
		
		cache_data, err := os.ReadFile(cache_file)
		if err != nil {
			return nil, err
		}

		// Unmarshal the cache data into the descriptor cache
		var descriptor_cache DescriptorCache
		err = json.Unmarshal(cache_data, &descriptor_cache)
		if err != nil {
			return nil, err
		}

		cached_descriptor_cache = &descriptor_cache
		return cached_descriptor_cache, nil
	}

	return nil, fmt.Errorf("invalid scroll path: %s", scroll_path)
}

// Looks up the cache for the scroll path and adds the function descriptor to it
func AddFunctionDescriptor(function_descriptor descriptor.FunctionDescriptor) error {
	// Calculate the hash of the scroll path
	scroll_path_hash, err := utils.HashFilePath(function_descriptor.ScrollPath)
	if err != nil {
		return err
	}

	// Look up the cache for the scroll path
	descriptor_cache, err := ReadDescriptorCache(function_descriptor.ScrollPath)
	if err != nil {
		return err
	}

	// Add the function descriptor to the cache
	descriptor_cache.Functions[function_descriptor.FunctionName] = function_descriptor

	// Update the scroll hash. Scroll here refers to the scroll.yaml file that this file caches for
	scroll_hash, err := utils.HashFile(function_descriptor.ScrollPath)
	if err != nil {
		return err
	}
	descriptor_cache.ScrollHash = scroll_hash

	// Write the cache to disk
	cache_path, err := cachePath()
	if err != nil {
		return err
	}
	cache_file := filepath.Join(cache_path, scroll_path_hash + ".json")
	cache_data, err := json.Marshal(descriptor_cache)
	if err != nil {
		return err
	}
	
	var indentedCacheData bytes.Buffer
	if err := json.Indent(&indentedCacheData, cache_data, "", "  "); err != nil {
		return err
	}
	err = os.WriteFile(cache_file, indentedCacheData.Bytes(), 0644)
	if err != nil {
		return err
	}
	return nil
}