// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package spawn

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
