package server

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveDuplicateTargetPath(t *testing.T) {
	t.Run("resolves valid file name in same directory", func(t *testing.T) {
		sourcePath := filepath.Join("C:\\", "A3", "config.txt")
		targetPath, err := resolveDuplicateTargetPath(sourcePath, "config (copy).txt")
		if err != nil {
			t.Fatalf("resolveDuplicateTargetPath() error = %v", err)
		}

		expected := filepath.Clean(filepath.Join("C:\\", "A3", "config (copy).txt"))
		if targetPath != expected {
			t.Fatalf("targetPath = %q, want %q", targetPath, expected)
		}
	})

	t.Run("rejects traversal name", func(t *testing.T) {
		sourcePath := filepath.Join("C:\\", "A3", "config.txt")
		_, err := resolveDuplicateTargetPath(sourcePath, "..\\config-copy.txt")
		if err == nil {
			t.Fatal("expected traversal name to fail")
		}
	})

	t.Run("rejects empty name", func(t *testing.T) {
		sourcePath := filepath.Join("C:\\", "A3", "config.txt")
		_, err := resolveDuplicateTargetPath(sourcePath, "   ")
		if err == nil {
			t.Fatal("expected empty name to fail")
		}
	})
}

func TestDuplicateFileContents(t *testing.T) {
	tempDir := t.TempDir()
	sourcePath := filepath.Join(tempDir, "source.txt")
	targetPath := filepath.Join(tempDir, "source (copy).txt")

	sourceContent := []byte("omnihance duplicate test")
	if err := os.WriteFile(sourcePath, sourceContent, 0640); err != nil {
		t.Fatalf("os.WriteFile(source) error = %v", err)
	}

	fileEditor := osFileEditor{}
	if err := duplicateFileContents(fileEditor, sourcePath, targetPath, 0640); err != nil {
		t.Fatalf("duplicateFileContents() error = %v", err)
	}

	duplicatedContent, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("os.ReadFile(target) error = %v", err)
	}

	if string(duplicatedContent) != string(sourceContent) {
		t.Fatalf("duplicatedContent = %q, want %q", duplicatedContent, sourceContent)
	}
}

func TestDuplicateFileContentsFailsForMissingSource(t *testing.T) {
	tempDir := t.TempDir()
	fileEditor := osFileEditor{}
	err := duplicateFileContents(
		fileEditor,
		filepath.Join(tempDir, "missing.txt"),
		filepath.Join(tempDir, "copy.txt"),
		0644,
	)
	if err == nil {
		t.Fatal("expected duplicateFileContents to fail for missing source")
	}
}

type osFileEditor struct{}

func (osFileEditor) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

func (osFileEditor) WriteFile(name string, data []byte, perm fs.FileMode) error {
	return os.WriteFile(name, data, perm)
}
