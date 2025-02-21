package testkube

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/kubeshop/testkube/pkg/utils"
)

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
	if w.Labels != nil {
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
	if w.Spec.Execution != nil {
		w.Spec.Execution.Tags = convertDotsInMap(w.Spec.Execution.Tags, fn)
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

func (w TestWorkflow) GetName() string {
	return w.Name
}

func (w TestWorkflow) GetNamespace() string {
	return w.Namespace
}

func (w TestWorkflow) GetLabels() map[string]string {
	return w.Labels
}

func (w TestWorkflow) GetAnnotations() map[string]string {
	return w.Annotations
}

func (w TestWorkflow) HasService(name string) bool {
	if w.Spec == nil {
		return false
	}

	steps := append(w.Spec.Setup, append(w.Spec.Steps, w.Spec.After...)...)
	for _, step := range steps {
		if step.HasService(name) {
			return true
		}
	}

	if _, ok := w.Spec.Services[name]; ok {
		return true
	}

	return false
}

func (w *TestWorkflow) DeepCopy() *TestWorkflow {
	if w == nil {
		return nil
	}
	v, _ := json.Marshal(w)
	var result TestWorkflow
	_ = json.Unmarshal(v, &result)
	return &result
}

// TODO: do it stable
func (w *TestWorkflow) Equals(other *TestWorkflow) bool {
	// Avoid check when there is one existing and the other one not
	if (w == nil) != (other == nil) {
		return false
	}

	// Reset timestamps to avoid influence
	wCreated := w.Created
	otherCreated := other.Created
	w.Created = time.Time{}
	other.Created = time.Time{}

	// Compare
	w1, _ := json.Marshal(w)
	w.Created = time.Time{}
	w2, _ := json.Marshal(other)
	result := bytes.Equal(w1, w2)

	// Restore values
	w.Created = wCreated
	other.Created = otherCreated

	return result
}
