// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetShardValues_LessThanCount(t *testing.T) {
	values := map[string][]interface{}{"a": {5, 10, 15}}
	assert.Equal(t, map[string][]interface{}{"a": {5}}, GetShardValues(values, 0, 5))
	assert.Equal(t, map[string][]interface{}{"a": {10}}, GetShardValues(values, 1, 5))
	assert.Equal(t, map[string][]interface{}{"a": {15}}, GetShardValues(values, 2, 5))
	assert.Equal(t, map[string][]interface{}{"a": {}}, GetShardValues(values, 3, 5))
	assert.Equal(t, map[string][]interface{}{"a": {}}, GetShardValues(values, 4, 5))
}

func TestGetShardValues_EqualCount(t *testing.T) {
	values := map[string][]interface{}{"a": {5, 10, 15}}
	assert.Equal(t, map[string][]interface{}{"a": {5}}, GetShardValues(values, 0, 3))
	assert.Equal(t, map[string][]interface{}{"a": {10}}, GetShardValues(values, 1, 3))
	assert.Equal(t, map[string][]interface{}{"a": {15}}, GetShardValues(values, 2, 3))
}

func TestGetShardValues_UnevenCount(t *testing.T) {
	values := map[string][]interface{}{"a": {5, 10, 15, 20, 25}}
	assert.Equal(t, map[string][]interface{}{"a": {5, 10}}, GetShardValues(values, 0, 3))
	assert.Equal(t, map[string][]interface{}{"a": {15, 20}}, GetShardValues(values, 1, 3))
	assert.Equal(t, map[string][]interface{}{"a": {25}}, GetShardValues(values, 2, 3))
}

func TestGetMatrixValues(t *testing.T) {
	matrix := map[string][]interface{}{
		"os":      {"linux", "windows"},
		"browser": {"chrome", "firefox", "safari"},
	}
	want := []map[string]interface{}{
		{"browser": "chrome", "os": "linux"},
		{"browser": "chrome", "os": "windows"},
		{"browser": "firefox", "os": "linux"},
		{"browser": "firefox", "os": "windows"},
		{"browser": "safari", "os": "linux"},
		{"browser": "safari", "os": "windows"},
	}
	require.Equal(t, int64(len(want)), CountCombinations(matrix))

	// Each worker computes its own combination, so every call has to map an index to the same one.
	for range 100 {
		for i := range want {
			require.Equal(t, want[i], GetMatrixValues(matrix, int64(i)), "index %d", i)
		}
	}
}
