package text

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestObfuscate(t *testing.T) {

	t.Run("test Obfuscate standard string, 2 chars from begining and end", func(t *testing.T) {
		in := "Some Long Token !@31209301293"
		out := ObfuscateLR(in, 2, 2)
		assert.Equal(t, "So*************************93", out)
	})

	t.Run("test Obfuscate short string too much chars left", func(t *testing.T) {
		in := "Short"
		out := ObfuscateLR(in, 20, 0)
		assert.Equal(t, "*****", out)
	})

	t.Run("test Obfuscate short string too much right", func(t *testing.T) {
		in := "Short"
		out := ObfuscateLR(in, 0, 20)
		assert.Equal(t, "*****", out)
	})
}
