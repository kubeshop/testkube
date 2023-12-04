package filesystem

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/kubeshop/testkube/pkg/utils/test"
)

func TestOSFileSystem_OpenFile_Integration(t *testing.T) {
	test.IntegrationTest(t)
	t.Parallel()

	// Create a temporary file and write some data to it
	f, err := os.CreateTemp("", "test.txt")
	if err != nil {
		t.Fatalf("failed to create temporary file: %v", err)
	}
	defer f.Close()
	defer os.Remove(f.Name())

	data := []byte("hello, world")
	if _, err := f.Write(data); err != nil {
		t.Fatalf("failed to write data to file: %v", err)
	}

	// Create a new instance of OSFileSystem and use it to read the temporary file
	fs := &OSFileSystem{}
	reader, err := fs.OpenFileBuffered(f.Name())
	if err != nil {
		t.Fatalf("failed to open file: %v", err)
	}

	// Read the content of the file
	content, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	// Check that the content matches what we wrote earlier
	if string(content) != string(data) {
		t.Errorf("expected content %q, but got %q", string(data), string(content))
	}
}

func TestOSFileSystem_Walk_Integration(t *testing.T) {
	test.IntegrationTest(t)
	t.Parallel()
	// Create a temporary directory and some files
	dir, err := os.MkdirTemp("", "test-dir")
	if err != nil {
		t.Fatalf("failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(dir)

	files := []string{"file1.txt", "file2.txt", "file3.txt"}
	for _, file := range files {
		if err := os.WriteFile(filepath.Join(dir, file), []byte("hello, world"), 0644); err != nil {
			t.Fatalf("failed to create file %q: %v", file, err)
		}
	}

	// Create a new instance of OSFileSystem and use it to walk the temporary directory
	fs := &OSFileSystem{}
	var fileList []string
	err = fs.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			fileList = append(fileList, path)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("failed to walk directory: %v", err)
	}

	// Check that the list of files matches what we created earlier
	expected := make(map[string]bool)
	for _, file := range files {
		expected[filepath.Join(dir, file)] = true
	}

	for _, file := range fileList {
		if !expected[file] {
			t.Errorf("unexpected file %q found in directory", file)
		}
	}
}

func TestOSFileSystem_OpenFileBuffered_Integration(t *testing.T) {
	test.IntegrationTest(t)
	t.Parallel()

	// Create a temporary file and write some data to it
	tempFile, err := os.CreateTemp("", "test_buffered_file.txt")
	if err != nil {
		t.Fatalf("failed to create temporary file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	if _, err := tempFile.WriteString("hello, world"); err != nil {
		t.Fatalf("failed to write to temporary file: %v", err)
	}
	if err := tempFile.Close(); err != nil {
		t.Fatalf("failed to close temporary file: %v", err)
	}

	// Open the temporary file with OSFileSystem and read it using OpenFileBuffered
	fs := &OSFileSystem{}
	reader, err := fs.OpenFileBuffered(tempFile.Name())
	if err != nil {
		t.Fatalf("failed to open file with OpenFileBuffered: %v", err)
	}

	// Check that the data read from the file matches what we wrote earlier
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("failed to read data from file: %v", err)
	}

	if string(data) != "hello, world" {
		t.Errorf("unexpected data read from file: got %q, expected %q", string(data), "hello, world")
	}
}

func TestOSFileSystem_Integration_ReadDir(t *testing.T) {
	test.IntegrationTest(t)
	t.Parallel()
	// Setup: Create a temporary directory
	tempDir, err := os.MkdirTemp("", "readDirTest")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %s", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a few files in the directory
	fileNames := []string{"file1.txt", "file2.txt", "file3.txt"}
	for _, fName := range fileNames {
		filePath := tempDir + "/" + fName
		if _, err := os.Create(filePath); err != nil {
			t.Fatalf("Failed to create file %s: %s", filePath, err)
		}
	}

	// Execution: Call ReadDir
	fs := NewOSFileSystem()
	entries, err := fs.ReadDir(tempDir)
	if err != nil {
		t.Errorf("ReadDir returned error: %s", err)
	}

	// Verification: Check if the returned entries match the created files
	if len(entries) != len(fileNames) {
		t.Errorf("Expected %d entries, got %d", len(fileNames), len(entries))
	}

	found := make(map[string]bool)
	for _, entry := range entries {
		found[entry.Name()] = true
	}

	for _, fName := range fileNames {
		if !found[fName] {
			t.Errorf("File %s was not found in directory entries", fName)
		}
	}
}

func TestOSFileSystem_Getwd(t *testing.T) {
	t.Parallel()

	// Create an instance of OSFileSystem
	fs := NewOSFileSystem()

	// Execution: Call Getwd
	wd, err := fs.Getwd()
	if err != nil {
		t.Fatalf("Getwd returned error: %s", err)
	}

	// Verification: Check if the returned working directory matches os.Getwd
	expectedWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd returned error: %s", err)
	}

	if wd != expectedWd {
		t.Errorf("Expected working directory '%s', got '%s'", expectedWd, wd)
	}
}
