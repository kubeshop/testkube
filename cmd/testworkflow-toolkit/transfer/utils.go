package transfer

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
)

func SourceID(dirPath string, files []string) string {
	v, err := json.Marshal(map[string]interface{}{"p": dirPath, "v": files})
	if err != nil {
		panic(fmt.Sprintf("error while serializing data for building checksum for transfer: %s", err.Error()))
	}
	return fmt.Sprintf("%x", sha256.Sum256(v))
}
