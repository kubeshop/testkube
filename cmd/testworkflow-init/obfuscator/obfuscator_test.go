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

	content1 := []byte("there is some sensitiv")
	content2 := []byte("e content in scope of testkube")
	n1, _ := passthrough.Write(content1)
	n2, _ := passthrough.Write(content2)

	result, err := io.ReadAll(buf)

	assert.NoError(t, err)
	assert.Equal(t, len(content1), n1)
	assert.Equal(t, len(content2), n2)
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

	content := []byte("sensitiv")
	n, _ := passthrough.Write(content)
	passthrough.Flush()

	result, err := io.ReadAll(buf)

	assert.NoError(t, err)
	assert.Equal(t, n, len(content))
	assert.Equal(t, "*****i*****", string(result))
}

func TestObfuscator_Order(t *testing.T) {
	buf := &bytes.Buffer{}
	passthrough := New(buf, FullReplace("*****"), []string{
		"sensitive",
		"sens",
	})

	content := []byte("there is some sensitive content in scope of testkube")
	n, _ := passthrough.Write(content)

	result, err := io.ReadAll(buf)

	assert.NoError(t, err)
	assert.Equal(t, n, len(content))
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
