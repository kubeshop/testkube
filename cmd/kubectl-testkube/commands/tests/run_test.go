package tests

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_readConfigFiles(t *testing.T) {
	t.Run("Files exist, files are properly read", func(t *testing.T) {
		files, err := createConfigFiles()
		assert.NoError(t, err)

		configFiles := []string{}
		for _, f := range files {
			configFiles = append(configFiles, fmt.Sprintf("%s:%s_b", f.Name(), f.Name()))
		}

		gotFiles, err := readConfigFiles(configFiles)
		assert.NoError(t, err)

		for _, f := range gotFiles {
			assert.Contains(t, string(f), "config file #")
		}

		err = cleanup(files)
		assert.NoError(t, err)
	})
	t.Run("Files don't exist, an error is thrown", func(t *testing.T) {
		configFiles := []string{"/tmp/file_does_not_exist:/path_not_important"}
		_, err := readConfigFiles(configFiles)
		assert.Error(t, err)
	})
}

func createConfigFiles() ([]*os.File, error) {
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
