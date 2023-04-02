//go:build integration

package runner

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

const repoURI = "https://github.com/kubeshop/testkube-executor-ginkgo.git"

func TestRun_Integration(t *testing.T) {
	ctx := context.Background()

	t.Run("GinkgoRunner should run tests from a repo that pass", func(t *testing.T) {
		runner, err := NewGinkgoRunner(ctx)

		if err != nil {
			t.Fail()
		}
		vars := make(map[string]testkube.Variable)
		variableOne := testkube.Variable{
			Name:  "GinkgoTestPackage",
			Value: "examples/e2e",
			Type_: testkube.VariableTypeBasic,
		}
		vars["GinkgoTestPackage"] = variableOne
		result, err := runner.Run(
			ctx,
			testkube.Execution{
				Content: &testkube.TestContent{
					Type_: string(testkube.TestContentTypeGitDir),
					Repository: &testkube.Repository{
						Type_:  "git",
						Uri:    repoURI,
						Branch: "main",
					},
				},
				Variables: vars,
			})

		assert.Equal(t, testkube.ExecutionStatusPassed, result.Status)
		assert.NoError(t, err)
	})

	t.Run("GinkgoRunner should run tests from a repo that fail", func(t *testing.T) {
		assert.NoError(t, os.Setenv("RUNNER_GITUSERNAME", "testuser"))
		assert.NoError(t, os.Setenv("RUNNER_GITTOKEN", "testtoken"))
		runner, err := NewGinkgoRunner(ctx)
		if err != nil {
			t.Fail()
		}
		vars := make(map[string]testkube.Variable)
		variableOne := testkube.Variable{
			Name:  "GinkgoTestPackage",
			Value: "examples/other",
			Type_: testkube.VariableTypeBasic,
		}
		vars["GinkgoTestPackage"] = variableOne
		result, err := runner.Run(
			ctx,
			testkube.Execution{
				Content: &testkube.TestContent{
					Type_: string(testkube.TestContentTypeGitDir),
					Repository: &testkube.Repository{
						Type_:  "git",
						Uri:    "https://github.com/kubeshop/testkube-executor-ginkgo.git",
						Branch: "main",
					},
				},
				Variables: vars,
			})

		assert.Equal(t, testkube.ExecutionStatusFailed, result.Status)
		assert.NoError(t, err)
	})
}
