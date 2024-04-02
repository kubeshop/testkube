// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package spawn

import "golang.org/x/exp/maps"

func CountCombinations(matrix map[string][]interface{}) int64 {
	combinations := int64(1)
	for k := range matrix {
		combinations *= int64(len(matrix[k]))
	}
	return combinations
}

func GetMatrixValues(matrix map[string][]interface{}, index int64) map[string]interface{} {
	// Compute modulo for each matrix parameter
	keys := maps.Keys(matrix)
	modulo := map[string]int64{}
	floor := map[string]int64{}
	for i, k := range keys {
		modulo[k] = int64(len(matrix[k]))
		floor[k] = 1
		for j := i + 1; j < len(keys); j++ {
			floor[k] *= int64(len(matrix[keys[j]]))
		}
	}

	// Compute values for selected index
	result := make(map[string]interface{})
	for _, k := range keys {
		kIdx := (index / floor[k]) % modulo[k]
		result[k] = matrix[k][kIdx]
	}
	return result
}

func GetShardValues(values map[string][]interface{}, index int64, count int64) map[string][]interface{} {
	result := make(map[string][]interface{})
	for k := range values {
		if index > int64(len(values[k])) {
			result[k] = []interface{}{}
			continue
		}
		size := count / int64(len(values[k]))
		if size == 0 {
			size = 1
		}
		start := index * size
		end := start + size
		if end > int64(len(values[k])) {
			result[k] = values[k][start:]
		} else {
			result[k] = values[k][start:end]
		}
	}
	return result
}
