// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

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
