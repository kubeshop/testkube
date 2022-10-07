package content

import (
	"fmt"
	"os"
)

func PlaceFiles(files map[string]string) error {
	for location, content := range files {
		err := os.WriteFile(location, []byte(content), 0644)
		if err != nil {
			return fmt.Errorf("could not write file: %w", err)
		}
	}
	return nil
}
