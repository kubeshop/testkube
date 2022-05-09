package v1

import (
	"testing"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/stretchr/testify/assert"
)

func TestParamsNilAssign(t *testing.T) {

	t.Run("merge two maps", func(t *testing.T) {

		p1 := map[string]string{"p1": "1"}
		p2 := map[string]testkube.Variable{"p2": testkube.NewBasicVariable("p2", "2")}

		out := mergeVariables(p1, p2)

		assert.Equal(t, 2, len(out))
		assert.Equal(t, "1", out["p1"].Value)
	})

	t.Run("merge with nil map", func(t *testing.T) {

		p2 := map[string]testkube.Variable{"p2": testkube.NewBasicVariable("p2", "2")}

		out := mergeVariables(nil, p2)

		assert.Equal(t, 1, len(out))
		assert.Equal(t, "1", out["p1"].Value)
	})

}
