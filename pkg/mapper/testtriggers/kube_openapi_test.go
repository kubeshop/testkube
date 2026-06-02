package testtriggers

import (
	"testing"

	"github.com/stretchr/testify/assert"

	testsv1 "github.com/kubeshop/testkube/api/testtriggers/v1"
)

func TestMapContentGitPullRequestFromCRD_NilInput(t *testing.T) {
	result := mapContentGitPullRequestFromCRD(nil)
	assert.Nil(t, result)
}

func TestMapContentGitPullRequestFromCRD_CopiesAllFields(t *testing.T) {
	pr := &testsv1.TestTriggerContentGitPullRequest{
		Types:          []string{"opened", "reopened"},
		Branches:       []string{"main"},
		BranchesIgnore: []string{"release/legacy-*"},
	}
	result := mapContentGitPullRequestFromCRD(pr)
	assert.NotNil(t, result)
	assert.Equal(t, []string{"opened", "reopened"}, result.Types)
	assert.Equal(t, []string{"main"}, result.Branches)
	assert.Equal(t, []string{"release/legacy-*"}, result.BranchesIgnore)
}
