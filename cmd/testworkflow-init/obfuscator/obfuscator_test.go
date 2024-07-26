package obfuscator

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestObfuscator_Full(t *testing.T) {
	buf := &bytes.Buffer{}
	passthrough := New(buf, FullReplace("*****"), []string{
		"sensitive",
		"scope",
		"testKube",
	})

	_, _ = passthrough.Write([]byte("there is some sensitive content in scope of testkube"))

	result, err := io.ReadAll(buf)

	assert.NoError(t, err)
	assert.Equal(t, "there is some ***** content in ***** of testkube", string(result))
}

func TestObfuscator_End(t *testing.T) {
	buf := &bytes.Buffer{}
	passthrough := New(buf, FullReplace("*****"), []string{
		"sensitive",
		"scope",
		"testKube",
	})

	_, _ = passthrough.Write([]byte("there is some sensitive"))

	result, err := io.ReadAll(buf)

	assert.NoError(t, err)
	assert.Equal(t, "there is some *****", string(result))
}

func TestObfuscator_Partial(t *testing.T) {
	buf := &bytes.Buffer{}
	passthrough := New(buf, FullReplace("*****"), []string{
		"sensitive",
		"scope",
	})

	_, _ = passthrough.Write([]byte("there is some sensitiv"))
	_, _ = passthrough.Write([]byte("e content in scope of testkube"))

	result, err := io.ReadAll(buf)

	assert.NoError(t, err)
	assert.Equal(t, "there is some ***** content in ***** of testkube", string(result))
}

func TestObfuscator_FlushLowerHit(t *testing.T) {
	buf := &bytes.Buffer{}
	passthrough := New(buf, FullReplace("*****"), []string{
		"sensitive",
		"sens",
	})

	_, _ = passthrough.Write([]byte("sensitiv"))
	passthrough.Flush()

	result, err := io.ReadAll(buf)

	assert.NoError(t, err)
	assert.Equal(t, "*****itiv", string(result))
}

func TestObfuscator_FlushNoHit(t *testing.T) {
	buf := &bytes.Buffer{}
	passthrough := New(buf, FullReplace("*****"), []string{
		"sensitive",
	})

	_, _ = passthrough.Write([]byte("sensitiv"))
	passthrough.Flush()

	result, err := io.ReadAll(buf)

	assert.NoError(t, err)
	assert.Equal(t, "sensitiv", string(result))
}

func TestObfuscator_FlushDoubleHit(t *testing.T) {
	buf := &bytes.Buffer{}
	passthrough := New(buf, FullReplace("*****"), []string{
		"sensitive",
		"sens",
		"tiv",
	})

	_, _ = passthrough.Write([]byte("sensitiv"))
	passthrough.Flush()

	result, err := io.ReadAll(buf)

	assert.NoError(t, err)
	assert.Equal(t, "*****i*****", string(result))
}

func TestObfuscator_Order(t *testing.T) {
	buf := &bytes.Buffer{}
	passthrough := New(buf, FullReplace("*****"), []string{
		"sensitive",
		"sens",
	})

	_, _ = passthrough.Write([]byte("there is some sensitive content in scope of testkube"))

	result, err := io.ReadAll(buf)

	assert.NoError(t, err)
	assert.Equal(t, "there is some ***** content in scope of testkube", string(result))
}

func TestObfuscator_Multiple(t *testing.T) {
	buf := &bytes.Buffer{}
	passthrough := New(buf, FullReplace("*****"), []string{
		"hello world",
		"hello",
		"blah",
	})

	_, _ = passthrough.Write([]byte("hello world there hello hahahaha helblah in there blah"))

	result, err := io.ReadAll(buf)

	assert.NoError(t, err)
	assert.Equal(t, "***** there ***** hahahaha hel***** in there *****", string(result))
}
