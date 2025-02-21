package testkube

import "github.com/kubeshop/testkube/pkg/utils"

func (t *TestWorkflowExecutionTags) ConvertDots(fn func(string) string) *TestWorkflowExecutionTags {
	if t.Tags != nil {
		t.Tags = convertDotsInMap(t.Tags, fn)
	}
	return t
}

func (t *TestWorkflowExecutionTags) EscapeDots() *TestWorkflowExecutionTags {
	return t.ConvertDots(utils.EscapeDots)
}

func (t *TestWorkflowExecutionTags) UnscapeDots() *TestWorkflowExecutionTags {
	return t.ConvertDots(utils.UnescapeDots)
}
