package testkube

import "github.com/kubeshop/testkube/pkg/utils"

type TestWorkflows []TestWorkflow

func (t TestWorkflows) Table() (header []string, output [][]string) {
	header = []string{"Name", "Description", "Created", "Labels"}
	for _, e := range t {
		output = append(output, []string{
			e.Name,
			e.Description,
			e.Created.String(),
			MapToString(e.Labels),
		})
	}

	return
}

func convertDotsInMap[T any](m map[string]T, fn func(string) string) map[string]T {
	result := make(map[string]T)
	for key, value := range m {
		result[fn(key)] = value
	}
	return result
}

func (w *TestWorkflow) ConvertDots(fn func(string) string) *TestWorkflow {
	if w == nil {
		return w
	}
	if w.Labels == nil {
		w.Labels = convertDotsInMap(w.Labels, fn)
	}
	if w.Spec.Pod != nil {
		w.Spec.Pod.Labels = convertDotsInMap(w.Spec.Pod.Labels, fn)
		w.Spec.Pod.Annotations = convertDotsInMap(w.Spec.Pod.Annotations, fn)
		w.Spec.Pod.NodeSelector = convertDotsInMap(w.Spec.Pod.NodeSelector, fn)
	}
	if w.Spec.Job != nil {
		w.Spec.Job.Labels = convertDotsInMap(w.Spec.Job.Labels, fn)
		w.Spec.Job.Annotations = convertDotsInMap(w.Spec.Job.Annotations, fn)
	}
	for i := range w.Spec.Use {
		if w.Spec.Use[i].Config != nil {
			w.Spec.Use[i].Config = convertDotsInMap(w.Spec.Use[i].Config, fn)
		}
	}
	for i := range w.Spec.Setup {
		w.Spec.Setup[i].ConvertDots(fn)
	}
	for i := range w.Spec.Steps {
		w.Spec.Steps[i].ConvertDots(fn)
	}
	for i := range w.Spec.After {
		w.Spec.After[i].ConvertDots(fn)
	}
	return w
}

func (w *TestWorkflow) EscapeDots() *TestWorkflow {
	return w.ConvertDots(utils.EscapeDots)
}

func (w *TestWorkflow) UnscapeDots() *TestWorkflow {
	return w.ConvertDots(utils.UnescapeDots)
}

func (w *TestWorkflow) GetObjectRef() *ObjectRef {
	return &ObjectRef{
		Name:      w.Name,
		Namespace: w.Namespace,
	}
}

func (w *TestWorkflow) QuoteWorkflowTextFields() {
}
