package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandUserPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir: %v", err)
	}

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "tilde_only",
			input: "~",
			want:  home,
		},
		{
			name:  "tilde_slash_path",
			input: "~/foo/bar",
			want:  filepath.Join(home, "foo", "bar"),
		},
		{
			name:  "tilde_slash_single_segment",
			input: "~/venv",
			want:  filepath.Join(home, "venv"),
		},
		{
			name:  "absolute_path",
			input: "/usr/bin/python",
			want:  "/usr/bin/python",
		},
		{
			name:  "relative_path",
			input: "venv/bin/python",
			want:  "venv/bin/python",
		},
		{
			name:  "empty_string",
			input: "",
			want:  "",
		},
		{
			// "~user" does not start with "~/", so no expansion occurs.
			// Only the exact prefix "~/" (or bare "~") is handled.
			name:  "tilde_no_slash",
			input: "~user",
			want:  "~user",
		},
		{
			name:  "dot_relative_path",
			input: "./script.py",
			want:  "./script.py",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ExpandUserPath(tc.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("ExpandUserPath(%q)\n  got:  %q\n  want: %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestHashFilePathAndContent(t *testing.T) {
	t.Run("same_path_gives_same_path_hash", func(t *testing.T) {
		f, err := os.CreateTemp("", "test_hash_*.txt")
		if err != nil {
			t.Fatalf("CreateTemp: %v", err)
		}
		f.WriteString("hello")
		f.Close()
		defer os.Remove(f.Name())

		h1, _, err := HashFilePathAndContent(f.Name())
		if err != nil {
			t.Fatalf("first call: %v", err)
		}
		h2, _, err := HashFilePathAndContent(f.Name())
		if err != nil {
			t.Fatalf("second call: %v", err)
		}
		if h1 != h2 {
			t.Errorf("same path produced different path hashes: %q vs %q", h1, h2)
		}
	})

	t.Run("different_paths_give_different_path_hashes", func(t *testing.T) {
		f1, err := os.CreateTemp("", "test_hash_a_*.txt")
		if err != nil {
			t.Fatalf("CreateTemp a: %v", err)
		}
		f1.WriteString("same content")
		f1.Close()
		defer os.Remove(f1.Name())

		f2, err := os.CreateTemp("", "test_hash_b_*.txt")
		if err != nil {
			t.Fatalf("CreateTemp b: %v", err)
		}
		f2.WriteString("same content")
		f2.Close()
		defer os.Remove(f2.Name())

		h1, _, err := HashFilePathAndContent(f1.Name())
		if err != nil {
			t.Fatalf("file a: %v", err)
		}
		h2, _, err := HashFilePathAndContent(f2.Name())
		if err != nil {
			t.Fatalf("file b: %v", err)
		}
		if h1 == h2 {
			t.Errorf("different paths produced the same path hash: %q", h1)
		}
	})

	t.Run("same_content_gives_same_content_hash", func(t *testing.T) {
		content := []byte("requirements==1.0.0\n")

		f1, err := os.CreateTemp("", "test_hash_c1_*.txt")
		if err != nil {
			t.Fatalf("CreateTemp c1: %v", err)
		}
		f1.Write(content)
		f1.Close()
		defer os.Remove(f1.Name())

		f2, err := os.CreateTemp("", "test_hash_c2_*.txt")
		if err != nil {
			t.Fatalf("CreateTemp c2: %v", err)
		}
		f2.Write(content)
		f2.Close()
		defer os.Remove(f2.Name())

		_, c1, err := HashFilePathAndContent(f1.Name())
		if err != nil {
			t.Fatalf("file c1: %v", err)
		}
		_, c2, err := HashFilePathAndContent(f2.Name())
		if err != nil {
			t.Fatalf("file c2: %v", err)
		}
		if c1 != c2 {
			t.Errorf("same content produced different content hashes: %q vs %q", c1, c2)
		}
	})

	t.Run("changed_content_gives_different_content_hash", func(t *testing.T) {
		f, err := os.CreateTemp("", "test_hash_d_*.txt")
		if err != nil {
			t.Fatalf("CreateTemp: %v", err)
		}
		f.WriteString("original content")
		f.Close()
		defer os.Remove(f.Name())

		_, before, err := HashFilePathAndContent(f.Name())
		if err != nil {
			t.Fatalf("hash before update: %v", err)
		}

		if err := os.WriteFile(f.Name(), []byte("changed content"), 0644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}

		_, after, err := HashFilePathAndContent(f.Name())
		if err != nil {
			t.Fatalf("hash after update: %v", err)
		}

		if before == after {
			t.Errorf("expected different content hashes after file change, but both are %q", before)
		}
	})

	t.Run("nonexistent_file_returns_error", func(t *testing.T) {
		_, _, err := HashFilePathAndContent("/tmp/nonexistent_sigil_hash_test_file.txt")
		if err == nil {
			t.Fatal("expected error for nonexistent file, got nil")
		}
	})
}
