package testworkflows

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetetTimestampLength(t *testing.T) {
	t.Run("returns length of nano for valid timestamps", func(t *testing.T) {
		l := getTimestampLength("2006-01-02T15:04:05.999999999Z07:00")
		assert.Equal(t, len(time.RFC3339Nano), l)

		l = getTimestampLength("2006-01-02T15:04:05.999999999+07:00")
		assert.Equal(t, len(time.RFC3339Nano), l)
	})

	t.Run("returns 0 for invalid timestamps", func(t *testing.T) {
		l := getTimestampLength("2006-01-02T15:04:05.99")
		assert.Equal(t, 0, l)

		l = getTimestampLength("2006-01-02T15:04:05.99")
		assert.Equal(t, 0, l)
	})
}

func TestLoadFilesFromDirectory(t *testing.T) {
	// Specify the directory containing the files
	d, _ := os.Getwd()
	fmt.Printf("DDDD: %+v\n", d)

	dir := filepath.Join(d, "testfiles")

	// Read the list of files from the directory
	files, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read directory: %v", err)
	}

	for _, file := range files {
		// Ensure the file is not a directory
		if file.IsDir() {
			continue
		}

		// Construct the full path of the file
		filePath := filepath.Join(dir, file.Name())

		// Open the file
		f, err := os.Open(filePath)
		if err != nil {
			t.Fatalf("failed to open file %s: %v", file.Name(), err)
		}

		// Read the file content
		content, err := io.ReadAll(f)
		f.Close() // Close the file after reading
		if err != nil {
			t.Fatalf("failed to read file %s: %v", file.Name(), err)
		}

		// Do something with the content, e.g., ensure it's not empty
		if len(content) == 0 {
			t.Errorf("file %s is empty", file.Name())
			t.Fail()
		}

		tr := true
		printStructuredLogLines(string(content), &tr)

	}

	fmt.Printf("%+v\n", len("2024-09-06T11:20:30.81675463"))

	t.Fail()
}
