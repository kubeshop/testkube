package tests

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_readCopyFiles(t *testing.T) {
	t.Run("Files exist, files are properly read", func(t *testing.T) {
		files, err := createCopyFiles()
		assert.NoError(t, err)

		copyFiles := []string{}
		for _, f := range files {
			copyFiles = append(copyFiles, fmt.Sprintf("%s:%s_b", f.Name(), f.Name()))
		}

		gotFiles, err := readCopyFiles(copyFiles)
		assert.NoError(t, err)

		for _, f := range gotFiles {
			assert.Contains(t, f, "config file #")
		}

		err = cleanup(files)
		assert.NoError(t, err)
	})
	t.Run("Files don't exist, an error is thrown", func(t *testing.T) {
		copyFiles := []string{"/tmp/file_does_not_exist:/path_not_important"}
		_, err := readCopyFiles(copyFiles)
		assert.Error(t, err)
	})
}

func Test_mergeCopyFiles(t *testing.T) {
	t.Run("Two empty lists should return empty list", func(t *testing.T) {
		testFiles := []string{}
		executionFiles := []string{}

		result, err := mergeCopyFiles(testFiles, executionFiles)
		assert.NoError(t, err)
		assert.Empty(t, result)
	})
	t.Run("First list populated, second list empty should return first list", func(t *testing.T) {
		testFiles := []string{
			"/test/file:/tmp/test/file",
		}
		executionFiles := []string{}

		result, err := mergeCopyFiles(testFiles, executionFiles)
		assert.NoError(t, err)
		assert.Equal(t, testFiles, result)
	})
	t.Run("First list empty, second list populated should return second list", func(t *testing.T) {
		testFiles := []string{}
		executionFiles := []string{
			"/test/file:/tmp/test/file",
		}

		result, err := mergeCopyFiles(testFiles, executionFiles)
		assert.NoError(t, err)
		assert.Equal(t, executionFiles, result)
	})
	t.Run("Two populated lists with no overlapping should return merged list", func(t *testing.T) {
		testFiles := []string{
			"/test/file1:/tmp/test/file1",
			"/test/file2:/tmp/test/file2",
			"/test/file3:/tmp/test/file3",
		}
		executionFiles := []string{
			"/test/file4:/tmp/test/file4",
			"/test/file5:/tmp/test/file5",
			"/test/file6:/tmp/test/file6",
		}

		result, err := mergeCopyFiles(testFiles, executionFiles)
		assert.NoError(t, err)
		assert.Equal(t, 6, len(result))
	})
	t.Run("Two populated lists with one overlapping element should return merged list with no duplicates", func(t *testing.T) {
		testFiles := []string{
			"/test/file1:/tmp/test/file1",
			"/test/file2:/tmp/test/file2",
			"/test/file3:/tmp/test/file3",
		}
		executionFiles := []string{
			"/test/file4:/tmp/test/file4",
			"/test/file5:/tmp/test/file5",
			"/test/file1:/tmp/test/file1",
		}

		result, err := mergeCopyFiles(testFiles, executionFiles)
		assert.NoError(t, err)
		assert.Equal(t, 5, len(result))
	})
}

func createCopyFiles() ([]*os.File, error) {
	files := []*os.File{}
	for i := 0; i < 5; i++ {
		file, err := os.CreateTemp("/tmp", fmt.Sprintf("config_%d", i))
		if err != nil {
			return files, err
		}

		_, err = file.WriteString(fmt.Sprintf("config file #%d", i))
		if err != nil {
			return files, err
		}
		files = append(files, file)
	}
	return files, nil
}

func cleanup(files []*os.File) error {
	for _, f := range files {
		err := os.Remove(f.Name())
		if err != nil {
			return err
		}
	}
	return nil
}
