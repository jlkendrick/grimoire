package utils

import (
	"os"
	"io"
	"bytes"
	"strings"
	"encoding/hex"
	"crypto/sha256"
	"path/filepath"
)

// ExpandUserPath replaces a leading "~" or "~/" with the current user's home
// directory. Go does not expand shell tildes; paths like "~/foo" are literal.
func ExpandUserPath(path string) (string, error) {
	if path == "~" {
		return os.UserHomeDir()
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, path[2:]), nil
	}
	return path, nil
}

func HashFilePathAndContent(path string) (string,string, error) {
	// Hash the file path (path of the dependency file used to build the venv)
	path_hash := sha256.New()
	if _, err := io.Copy(path_hash, bytes.NewReader([]byte(path))); err != nil {
		return "", "", err
	}

	// Hash the file content (content of the dependency file used to build the venv)
	content_hash := sha256.New()
	content, err := os.Open(path)
	if err != nil {
		return "", "", err
	}
	defer content.Close()
	if _, err := io.Copy(content_hash, content); err != nil {
		return "", "", err
	}
	return hex.EncodeToString(path_hash.Sum(nil)), hex.EncodeToString(content_hash.Sum(nil)), nil
}

func UpwardsTraversalForTargets(start_dir string, target_files []string) (map[string]string, bool) {

	matched_targets := map[string]string{}

	for _, target_file := range target_files {
		// Check if the target file exists
		search_path := filepath.Join(start_dir, target_file)
		if _, err := os.Stat(search_path); err == nil {
			matched_targets[target_file] = search_path
		}
	}
	
	// If we have results, then we are done and can return
	if len(matched_targets) > 0 {
		return matched_targets, true
	}

	// If we don't have results, then we need to search the parent directory
	parent_dir := filepath.Dir(start_dir)
	if parent_dir == start_dir {
		return matched_targets, false
	}

	// Recursively search the parent directory
	matched_targets, found := UpwardsTraversalForTargets(parent_dir, target_files)
	if !found {
		return matched_targets, false
	}
	return matched_targets, true
}

func MakeRelativePath(path string, base string) (string, error) {
	rel, err := filepath.Rel(base, path)
	if err != nil {
		return "", err
	}
	return rel, nil
}