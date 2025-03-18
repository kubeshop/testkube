// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//     https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package testworkflows

// CountMapBytes returns the total bytes of the map
func CountMapBytes(m map[string]string) int {
	totalBytes := 0
	for k, v := range m {
		totalBytes += len(k) + len(v)
	}
	return totalBytes
}
