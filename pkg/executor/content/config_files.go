package content

import (
	"fmt"
	"os"
)

func PlaceFiles(files map[string][]byte) error {
	for location, content := range files {
		err := os.WriteFile(location, content, 0644)
		if err != nil {
			return fmt.Errorf("could not write file: %w", err)
		}
	}
	return nil
}
