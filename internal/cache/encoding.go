package cache

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"

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
	Functions  map[string]descriptor.FunctionDescriptor `json:"functions"` // spell command -> function descriptor
	Pipelines  map[string]descriptor.PipelineDescriptor `json:"pipelines"` // ritual command -> pipeline descriptor
}

// cached_descriptor_caches memoizes per-scroll DescriptorCache objects within
// a single CLI invocation, keyed by scroll path. Cobra command execution is
// single-threaded, so a plain map is safe.
var cached_descriptor_caches = map[string]*DescriptorCache{}

func ResetCache() {
	cached_descriptor_caches = map[string]*DescriptorCache{}
}

func cachePath() (string, error) {
	grimoire_home, err := utils.GrimoireHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(grimoire_home, "cache"), nil
}

// ReadDescriptorCache returns the DescriptorCache for one scroll, identified
// by its absolute scroll path. If no cache file exists yet, an empty
// DescriptorCache is created in memory and returned.
func ReadDescriptorCache(scroll_path string) (*DescriptorCache, error) {
	if cached, ok := cached_descriptor_caches[scroll_path]; ok {
		return cached, nil
	}

	cache_dir, err := cachePath()
	if err != nil {
		return nil, err
	}

	filename_hash, err := utils.HashFilePath(scroll_path)
	if err != nil {
		return nil, err
	}
	cache_file := filepath.Join(cache_dir, filename_hash+".json")

	if _, err := os.Stat(cache_file); os.IsNotExist(err) {
		dc := &DescriptorCache{
			Version:    CACHE_VERSION,
			ScrollHash: "", // empty so that we correctly trigger a re-run of the reconciler
			ScrollPath: scroll_path,
			Functions:  make(map[string]descriptor.FunctionDescriptor),
		}
		cached_descriptor_caches[scroll_path] = dc
		return dc, nil
	}

	cache_data, err := os.ReadFile(cache_file)
	if err != nil {
		return nil, err
	}
	var dc DescriptorCache
	if err := json.Unmarshal(cache_data, &dc); err != nil {
		return nil, err
	}
	cached_descriptor_caches[scroll_path] = &dc
	return &dc, nil
}

// AddFunctionDescriptor writes function_descriptor into the per-scroll cache
// identified by function_descriptor.ScrollPath, refreshing both the in-memory
// map entry and the on-disk JSON file.
func AddFunctionDescriptor(function_descriptor descriptor.FunctionDescriptor) error {
	descriptor_cache, err := ReadDescriptorCache(function_descriptor.ScrollPath)
	if err != nil {
		return err
	}

	descriptor_cache.Functions[function_descriptor.CommandName] = function_descriptor

	return WriteDescriptorCache(descriptor_cache)
}

// WriteDescriptorCache persists descriptor_cache to disk, refreshing
// ScrollHash from the current scroll file. Use this after mutating the
// in-memory cache (e.g. pruning entries) when the per-descriptor write
// path of AddFunctionDescriptor doesn't apply.
func WriteDescriptorCache(descriptor_cache *DescriptorCache) error {
	scroll_hash, err := utils.HashFile(descriptor_cache.ScrollPath)
	if err != nil {
		return err
	}
	descriptor_cache.ScrollHash = scroll_hash

	cache_dir, err := cachePath()
	if err != nil {
		return err
	}
	scroll_path_hash, err := utils.HashFilePath(descriptor_cache.ScrollPath)
	if err != nil {
		return err
	}
	cache_file := filepath.Join(cache_dir, scroll_path_hash+".json")

	cache_data, err := json.Marshal(descriptor_cache)
	if err != nil {
		return err
	}
	var indented bytes.Buffer
	if err := json.Indent(&indented, cache_data, "", "  "); err != nil {
		return err
	}
	return os.WriteFile(cache_file, indented.Bytes(), 0644)
}
