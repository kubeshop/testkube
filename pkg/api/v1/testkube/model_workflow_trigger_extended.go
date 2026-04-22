package testkube

import "fmt"

type WorkflowTriggers []WorkflowTrigger

func (list WorkflowTriggers) Table() (header []string, output [][]string) {
	header = []string{"Name", "Namespace", "Resource", "Event", "Workflow", "Disabled"}
	for _, t := range list {
		resource := ""
		if t.Watch != nil && t.Watch.Resource.Kind != "" {
			r := t.Watch.Resource
			if r.Group == "" {
				resource = fmt.Sprintf("%s/%s", r.Version, r.Kind)
			} else {
				resource = fmt.Sprintf("%s/%s/%s", r.Group, r.Version, r.Kind)
			}
		}
		workflow := t.Run.Workflow.Name
		if workflow == "" && t.Run.Workflow.NameRegex != "" {
			workflow = "~" + t.Run.Workflow.NameRegex
		}
		output = append(output, []string{
			t.Name, t.Namespace, resource, t.When.Event, workflow, fmt.Sprint(t.Disabled),
		})
	}
	return
}
