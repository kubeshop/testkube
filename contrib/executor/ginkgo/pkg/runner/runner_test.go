package runner

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func TestRun(t *testing.T) {
	t.Parallel()

	t.Run("FindGoinkgoParams should override default params when provided with new value", func(t *testing.T) {
		t.Parallel()

		defaultParams := InitializeGinkgoParams()
		variables := make(map[string]testkube.Variable)
		variableOne := testkube.Variable{
			Name:  "GinkgoTestPackage",
			Value: "e2e",
			Type_: testkube.VariableTypeBasic,
		}
		variableTwo := testkube.Variable{
			Name:  "GinkgoRecursive",
			Value: "",
			Type_: testkube.VariableTypeBasic,
		}
		variables["GinkgoTestPackage"] = variableOne
		variables["GinkgoRecursive"] = variableTwo
		execution := testkube.Execution{
			Variables: variables,
		}
		mappedParams := FindGinkgoParams(&execution, defaultParams)
		assert.Equal(t, "e2e", mappedParams["GinkgoTestPackage"])
		assert.Equal(t, "", mappedParams["GinkgoRecursive"])
	})
}
