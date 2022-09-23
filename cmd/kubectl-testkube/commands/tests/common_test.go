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
			assert.Contains(t, string(f), "config file #")
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
