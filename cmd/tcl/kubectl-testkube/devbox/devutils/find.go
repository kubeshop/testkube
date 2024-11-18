// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package devutils

import (
	"os"
	"path/filepath"
)

func FindDirContaining(paths ...string) string {
	cwd, _ := os.Getwd()

	// Find near in the tree
	current := filepath.Clean(filepath.Join(cwd, "testkube", "dummy"))
loop:
	for current != filepath.Clean(filepath.Join(cwd, "..")) {
		current = filepath.Dir(current)
		for _, path := range paths {
			expected := filepath.Clean(filepath.Join(current, path))
			_, err := os.Stat(expected)
			if err != nil {
				continue loop
			}
		}
		return current
	}
	return ""
}
