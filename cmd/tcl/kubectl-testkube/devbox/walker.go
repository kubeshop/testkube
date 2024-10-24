// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package devbox

import (
	"os"
	"path/filepath"
)

func findFile(path string) string {
	cwd, _ := os.Getwd()

	// Find near in the tree
	current := filepath.Clean(filepath.Join(cwd, "testkube"))
	for current != filepath.Clean(filepath.Join(cwd, "..")) {
		expected := filepath.Clean(filepath.Join(current, path))
		_, err := os.Stat(expected)
		if err == nil {
			return expected
		}
		current = filepath.Dir(current)
	}
	return ""
}
