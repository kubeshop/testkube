package resourcepattern

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResourcePattern_ParseDefault(t *testing.T) {
	pattern, _ := New("")
	r1, ok1 := pattern.Parse("something-here", nil)
	r2, ok2 := pattern.Parse("whatever-it-is-called", nil)
	r3, ok3 := pattern.Parse("whatever/blah/blah", nil)
	assert.True(t, ok1)
	assert.Equal(t, "something-here", r1.Name)
	assert.True(t, ok2)
	assert.Equal(t, "whatever-it-is-called", r2.Name)
	assert.False(t, ok3)
	assert.Nil(t, r3)
}

func TestResourcePattern_ParseDuplicate(t *testing.T) {
	pattern, _ := New("<name>-<name>")
	r1, ok1 := pattern.Parse("something-something", nil)
	r2, ok2 := pattern.Parse("something-something2", nil)
	assert.True(t, ok1)
	assert.Equal(t, "something", r1.Name)
	assert.Equal(t, "", r1.Generic["namespace"])
	assert.False(t, ok2)
	assert.Nil(t, r2)
}

func TestResourcePattern_ParseMultipleArguments(t *testing.T) {
	pattern, _ := New("<name>-<namespace>")
	r1, ok1 := pattern.Parse("something-something", nil)
	r2, ok2 := pattern.Parse("something-something2", nil)
	assert.True(t, ok1)
	assert.Equal(t, "something", r1.Name)
	assert.Equal(t, "something", r1.Generic["namespace"])
	assert.True(t, ok2)
	assert.Equal(t, "something", r2.Name)
	assert.Equal(t, "something2", r2.Generic["namespace"])
}

func TestResourcePattern_ParseGenericFilter(t *testing.T) {
	pattern, _ := New("<name>-<namespace>-<cluster>")
	r1, ok1 := pattern.Parse("something-something-hello", nil)
	r2, ok2 := pattern.Parse("something-something2-", nil)
	assert.True(t, ok1)
	assert.Equal(t, "something", r1.Name)
	assert.Equal(t, "something", r1.Generic["namespace"])
	assert.Equal(t, map[string]string{"cluster": "hello"}, r1.Generic)
	assert.False(t, ok2)
	assert.Nil(t, r2)
}

func TestResourcePattern_CompileDefault(t *testing.T) {
	pattern, _ := New("")
	r1, ok1 := pattern.Compile(&Metadata{Name: "some-name"})
	r2, ok2 := pattern.Compile(&Metadata{Name: "another-name", Generic: map[string]string{"namespace": "xyz"}})
	_, ok3 := pattern.Compile(&Metadata{Name: "another-name/blah", Generic: map[string]string{"namespace": "xyz"}})
	assert.True(t, ok1)
	assert.Equal(t, "some-name", r1)
	assert.True(t, ok2)
	assert.Equal(t, "another-name", r2)
	assert.False(t, ok3)
}

func TestResourcePattern_CompileDuplicate(t *testing.T) {
	pattern, _ := New("<name>/<name>")
	r1, ok1 := pattern.Compile(&Metadata{Name: "some-name"})
	assert.True(t, ok1)
	assert.Equal(t, "some-name/some-name", r1)
}

func TestResourcePattern_CompileMultipleArguments(t *testing.T) {
	pattern, _ := New("<namespace>/<name>")
	_, ok1 := pattern.Compile(&Metadata{Name: "some-name"})
	r2, ok2 := pattern.Compile(&Metadata{Name: "another-name", Generic: map[string]string{"namespace": "xyz"}})
	assert.False(t, ok1)
	assert.True(t, ok2)
	assert.Equal(t, "xyz/another-name", r2)
}

func TestResourcePattern_CompileGenericData(t *testing.T) {
	pattern, _ := New("<cluster>/<namespace>/<name>")
	_, ok1 := pattern.Compile(&Metadata{Name: "another-name", Generic: map[string]string{"namespace": "xyz"}})
	r2, ok2 := pattern.Compile(&Metadata{Name: "another-name", Generic: map[string]string{"namespace": "xyz", "cluster": "magic"}})
	assert.False(t, ok1)
	assert.True(t, ok2)
	assert.Equal(t, "magic/xyz/another-name", r2)
}
