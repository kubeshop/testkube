package obfuscator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFullReplace(t *testing.T) {
	assert.Equal(t, []byte("abc"), FullReplace("abc")([]byte("def")))
}

func TestLastCharacters(t *testing.T) {
	assert.Equal(t, []byte("***"), ShowLastCharacters("***", 0)([]byte("def")))
	assert.Equal(t, []byte("***f"), ShowLastCharacters("***", 1)([]byte("def")))
	assert.Equal(t, []byte("***ef"), ShowLastCharacters("***", 2)([]byte("def")))
	assert.Equal(t, []byte("def"), ShowLastCharacters("***", 3)([]byte("def")))
	assert.Equal(t, []byte("def"), ShowLastCharacters("***", 4)([]byte("def")))
}
