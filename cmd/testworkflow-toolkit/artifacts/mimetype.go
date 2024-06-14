// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package artifacts

import (
	"path/filepath"

	"github.com/h2non/filetype"
)

func DetectMimetype(filePath string) string {
	ext := filepath.Ext(filePath)

	// Remove the dot from the file extension
	if len(ext) > 0 && ext[0] == '.' {
		ext = ext[1:]
	}
	t := filetype.GetType(ext)
	if t == filetype.Unknown {
		return ""
	}
	return t.MIME.Value
}
