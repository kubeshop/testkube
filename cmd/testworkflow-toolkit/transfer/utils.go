package transfer

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
)

func SourceID(dirPath string, files []string) string {
	v, _ := json.Marshal(map[string]interface{}{"p": dirPath, "v": files})
	return fmt.Sprintf("%x", sha256.Sum256(v))
}
