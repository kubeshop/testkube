package text

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type stringerTestImpl struct {
	name string
}

func (tt stringerTestImpl) String() string {
	return tt.name
}

func TestToStr(t *testing.T) {

	t.Run("converts stringer to string", func(t *testing.T) {
		test := stringerTestImpl{name: "test"}
		assert.Equal(t, "test", ToStr(test))
	})

	t.Run("converts int to str", func(t *testing.T) {
		test := 1
		assert.Equal(t, "1", ToStr(test))
	})

	t.Run("converts float to str", func(t *testing.T) {
		test := 1.1
		assert.Equal(t, "1.1", ToStr(test))
	})

	t.Run("converts bool to str", func(t *testing.T) {
		test := true
		assert.Equal(t, "true", ToStr(test))
	})

	t.Run("converts nil to str", func(t *testing.T) {
		var test *string
		assert.Equal(t, "<nil>", ToStr(test))
	})

}
