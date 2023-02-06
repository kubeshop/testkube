package testkube

import "fmt"

func (e *ExecutorUpsertRequest) QuoteExecutorTextFields() {
	if e.JobTemplate != "" {
		e.JobTemplate = fmt.Sprintf("%q", e.JobTemplate)
	}

	for i := range e.Command {
		if e.Command[i] != "" {
			e.Command[i] = fmt.Sprintf("%q", e.Command[i])
		}
	}

	for i := range e.Args {
		if e.Args[i] != "" {
			e.Args[i] = fmt.Sprintf("%q", e.Args[i])
		}
	}

	if e.Meta != nil {
		for key, value := range e.Meta.Tooltips {
			if value != "" {
				e.Meta.Tooltips[key] = fmt.Sprintf("%q", value)
			}
		}
	}
}
