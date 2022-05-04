package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParamsNilAssign(t *testing.T) {

	t.Run("merge two maps", func(t *testing.T) {

		p1 := map[string]string{"p1": "1"}
		p2 := map[string]string{"p2": "2"}

		out := mergeVariables(p1, p2)

		assert.Equal(t, map[string]string{"p1": "1", "p2": "2"}, out)
	})

	t.Run("merge with nil map", func(t *testing.T) {

		p2 := map[string]string{"p2": "2"}

		out := mergeVariables(nil, p2)

		assert.Equal(t, map[string]string{"p2": "2"}, out)
	})

}
