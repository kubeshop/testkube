package runner

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func TestRun(t *testing.T) {
	t.Parallel()

	t.Run("InitializeGinkgoParams should should set up some default parameters for ginkgo", func(t *testing.T) {
		t.Parallel()

		defaultParams := InitializeGinkgoParams()
		assert.Equal(t, "", defaultParams["GinkgoTestPackage"])
		assert.Equal(t, "-r", defaultParams["GinkgoRecursive"])
		assert.Equal(t, "-p", defaultParams["GinkgoParallel"])
		assert.Equal(t, "--randomize-all", defaultParams["GinkgoRandomize"])
		assert.Equal(t, "--randomize-suites", defaultParams["GinkgoRandomizeSuites"])
		assert.Equal(t, "--trace", defaultParams["GinkgoTrace"])
		assert.Equal(t, "--junit-report report.xml", defaultParams["GinkgoJunitReport"])

	})

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

	t.Run("BuildGinkgoArgs should build ginkgo args slice", func(t *testing.T) {
		t.Parallel()

		defaultParams := InitializeGinkgoParams()
		argSlice, err := BuildGinkgoArgs(defaultParams, "", "")
		assert.Nil(t, err)
		assert.Contains(t, argSlice, "-r")
		assert.Contains(t, argSlice, "-p")
		assert.Contains(t, argSlice, "--randomize-all")
		assert.Contains(t, argSlice, "--randomize-suites")
		assert.Contains(t, argSlice, "--trace")
		assert.Contains(t, argSlice, "--junit-report")
		assert.Contains(t, argSlice, "report.xml")
	})

	t.Run("BuildGinkgoPassThroughFlags should build pass through flags slice from leftover Variables and from Args", func(t *testing.T) {
		t.Parallel()

		variables := make(map[string]testkube.Variable)
		variableOne := testkube.Variable{
			Name:  "one",
			Value: "one",
			Type_: testkube.VariableTypeBasic,
		}
		variableTwo := testkube.Variable{
			Name:  "two",
			Value: "two",
			Type_: testkube.VariableTypeBasic,
		}
		variables["GinkgoPassThroughOne"] = variableOne
		variables["GinkgoPassThroughTwo"] = variableTwo

		args := []string{
			"--three",
			"--four=four",
		}

		execution := testkube.Execution{
			Variables: variables,
			Args:      args,
		}
		passThroughs := BuildGinkgoPassThroughFlags(execution)
		assert.Contains(t, passThroughs, "--")
		assert.Equal(t, os.Getenv("one"), "one")
		assert.Equal(t, os.Getenv("two"), "two")
		assert.Contains(t, passThroughs, "--three")
		assert.Contains(t, passThroughs, "--four=four")
	})
}
