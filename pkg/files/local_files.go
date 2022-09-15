package files

import (
	"fmt"
	"os"
)

type LocalFile struct{}

func GetContents(location string) (string, error) {
	content, err := os.ReadFile(location)
	if err != nil {
		return "", fmt.Errorf("could not read file from location %s: %w", location, err)
	}
	return string(content), nil
}
