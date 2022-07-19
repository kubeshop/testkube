package runner

import (
	"fmt"
	"os"
	"testing"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func TestRun(t *testing.T) {
	t.Skip("move this test to e2e test suite with valid environment setup")

	// Can't run it in my default install on mac
	os.Setenv("CYPRESS_CACHE_FOLDER", os.TempDir())

	runner, err := NewCypressRunner()
	if err != nil {
		t.Fail()
	}
	repoURI := "https://github.com/kubeshop/testkube-executor-cypress.git"
	result, err := runner.Run(testkube.Execution{
		Content: &testkube.TestContent{
			Type_: string(testkube.TestContentTypeGitDir),
			Repository: &testkube.Repository{
				Type_:  "git",
				Uri:    repoURI,
				Branch: "jacek/feature/json-output",
				Path:   "examples",
			},
		},
	})

	fmt.Printf("RESULT: %+v\n", result)
	fmt.Printf("ERROR:  %+v\n", err)

	t.Fail()

}
