package tests

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/api/v1/client"
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

// createTempFile creates temporary file with the given content
func createTempFile(content string) (*os.File, error) {
	file, err := os.CreateTemp("", "variables.txt")
	if err != nil {
		return nil, err
	}

	_, err = file.WriteString(content)
	if err != nil {
		return nil, err
	}

	return file, nil
}

// createLargeFile creates temporary file of the given size
func createLargeFile(size int64) (*os.File, error) {
	fd, err := os.CreateTemp("", "variables.txt")
	if err != nil {
		return nil, err
	}
	_, err = fd.Seek(size-1, 0)
	if err != nil {
		return nil, err
	}
	_, err = fd.Write([]byte{0})
	if err != nil {
		return nil, err
	}
	err = fd.Close()
	if err != nil {
		return nil, err
	}
	return fd, nil
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

func Test_PrepareVariablesFile(t *testing.T) {
	t.Run("File does not exist should return error", func(t *testing.T) {
		_, _, err := PrepareVariablesFile(client.APIClient{}, "parent", client.Test, "/this-file-does-not-exist", 0)
		assert.Error(t, err)
	})
	t.Run("File small enough should return contents", func(t *testing.T) {
		fileContent := "variables file"
		file, err := createTempFile(fileContent)
		assert.NoError(t, err)
		assert.NotEmpty(t, file)

		contents, isUploaded, err := PrepareVariablesFile(client.APIClient{}, "parent", client.Test, file.Name(), 0)
		assert.NoError(t, err)
		assert.False(t, isUploaded)
		assert.Equal(t, fileContent, contents)
	})
	t.Run("Big file should be uploaded", func(t *testing.T) {
		file, err := createLargeFile(maxArgSize + 1)
		assert.NoError(t, err)
		assert.NotEmpty(t, file)

		isCalled := false
		mockClient := client.APIClient{
			CopyFileClient: &client.MockCopyFileAPI{
				UploadFileFn: func(parentName string, parentType client.TestingType, filePath string, fileContent []byte, timeout time.Duration) error {
					isCalled = true
					return nil
				},
			},
		}
		path, isUploaded, err := PrepareVariablesFile(mockClient, "parent", client.Test, file.Name(), 0)
		assert.NoError(t, err)
		assert.True(t, isUploaded)
		assert.Contains(t, path, "variables.txt")
		assert.True(t, isCalled)
	})
}
