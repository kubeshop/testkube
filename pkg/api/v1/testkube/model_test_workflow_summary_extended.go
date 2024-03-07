package testkube

import (
	"github.com/kubeshop/testkube/pkg/utils"
)

func (w *TestWorkflowSummary) ConvertDots(fn func(string) string) *TestWorkflowSummary {
	if w == nil || w.Labels == nil {
		return w
	}
	if w.Labels != nil {
		w.Labels = convertDotsInMap(w.Labels, fn)
	}
	return w
}

func (w *TestWorkflowSummary) EscapeDots() *TestWorkflowSummary {
	return w.ConvertDots(utils.EscapeDots)
}

func (w *TestWorkflowSummary) UnscapeDots() *TestWorkflowSummary {
	return w.ConvertDots(utils.UnescapeDots)
}

func (w *TestWorkflowSummary) GetObjectRef() *ObjectRef {
	return &ObjectRef{
		Name:      w.Name,
		Namespace: w.Namespace,
	}
}

func (w *TestWorkflowSummary) QuoteWorkflowTextFields() {
}
