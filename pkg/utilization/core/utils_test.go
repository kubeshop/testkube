package core

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestReadLastLine(t *testing.T) {
	f, err := os.Open("testdata/metrics_valid_metadata.influx")
	require.NoError(t, err)
	t.Cleanup(func() {
		f.Close()
	})

	line, offset, err := readLastLine(f)
	require.NoError(t, err)

	println(offset)
	expectedLine := "#META lines=50 format=influx"
	assert.Equal(t, expectedLine, string(line))
	assert.Equal(t, 4430, offset)
}
