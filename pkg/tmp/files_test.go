package tmp

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReaderToTmpfile(t *testing.T) {
	const content = "test content"
	buffer := strings.NewReader(content)

	// when
	path, err := ReaderToTmpfile(buffer)

	// then
	assert.NoError(t, err)
	defer os.Remove(path) // clean up

	contentBytes, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}

	assert.Equal(t, content, string(contentBytes))

}
