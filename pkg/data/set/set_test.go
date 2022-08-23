package set

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOf(t *testing.T) {
	out := Of("aaa", "bbb")

	assert.True(t, out.Has("aaa"), "set should have aaa")
	assert.True(t, out.Has("bbb"), "set should have aaa")
}

func TestSet_ToArray(t *testing.T) {
	// given
	out := Of("aaa", "bbb")

	// when
	arr := out.ToArray()

	// then
	assert.Contains(t, arr, "aaa")
	assert.Contains(t, arr, "bbb")
}

func TestSet_Remove(t *testing.T) {
	// given
	out := Of("aaa", "bbb", "ccc")

	// when
	out.Remove("bbb")

	// then
	assert.False(t, out.Has("bbb"))
}
