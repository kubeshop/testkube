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
