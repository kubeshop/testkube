package testkube

import (
	"bytes"
	"encoding/json"
	"strings"
	"time"

	"github.com/kubeshop/testkube/pkg/utils"
)

type TestWorkflowTemplates []TestWorkflowTemplate

func (t TestWorkflowTemplates) Table() (header []string, output [][]string) {
	header = []string{"Name", "Description", "Created", "Labels"}
	for _, e := range t {
		output = append(output, []string{
			strings.ReplaceAll(e.Name, "--", "/"),
			e.Description,
			e.Created.String(),
			MapToString(e.Labels),
		})
	}

	return
}

func (w TestWorkflowTemplate) GetName() string {
	return w.Name
}

func (w TestWorkflowTemplate) GetNamespace() string {
	return w.Namespace
}

func (w TestWorkflowTemplate) GetLabels() map[string]string {
	return w.Labels
}

func (w TestWorkflowTemplate) GetAnnotations() map[string]string {
	return w.Annotations
}

func (w *TestWorkflowTemplate) EscapeDots() *TestWorkflowTemplate {
	return w.ConvertDots(utils.EscapeDots)
}

func (w *TestWorkflowTemplate) UnscapeDots() *TestWorkflowTemplate {
	return w.ConvertDots(utils.UnescapeDots)
}

func (w *TestWorkflowTemplate) ConvertDots(fn func(string) string) *TestWorkflowTemplate {
	if w == nil {
		return w
	}
	if w.Labels != nil {
		w.Labels = convertDotsInMap(w.Labels, fn)
	}
	if w.Annotations != nil {
		w.Annotations = convertDotsInMap(w.Annotations, fn)
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
	if w.Spec.Execution != nil {
		w.Spec.Execution.Tags = convertDotsInMap(w.Spec.Execution.Tags, fn)
	}

	for _, ev := range w.Spec.Events {
		if ev.Cronjob != nil {
			ev.Cronjob.Annotations = convertDotsInMap(ev.Cronjob.Annotations, fn)
			ev.Cronjob.Labels = convertDotsInMap(ev.Cronjob.Labels, fn)
		}

	}

	return w
}

func (w *TestWorkflowTemplate) DeepCopy() *TestWorkflowTemplate {
	if w == nil {
		return nil
	}
	v, _ := json.Marshal(w)
	var result TestWorkflowTemplate
	_ = json.Unmarshal(v, &result)
	return &result
}

// TODO: do it stable
func (w *TestWorkflowTemplate) Equals(other *TestWorkflowTemplate) bool {
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
