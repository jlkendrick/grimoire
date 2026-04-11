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