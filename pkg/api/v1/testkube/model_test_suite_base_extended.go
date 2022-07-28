package testkube

import (
	"fmt"

	"github.com/kubeshop/testkube/pkg/data/set"
)

type TestSuites []TestSuite

func (tests TestSuites) Table() (header []string, output [][]string) {
	header = []string{"Name", "Description", "Steps", "Labels", "Schedule"}
	for _, e := range tests {
		output = append(output, []string{
			e.Name,
			e.Description,
			fmt.Sprintf("%d", len(e.Steps)),
			MapToString(e.Labels),
			e.Schedule,
		})
	}

	return
}

func (t TestSuite) GetObjectRef() *ObjectRef {
	return &ObjectRef{
		Name:      t.Name,
		Namespace: t.Namespace,
	}
}

// GetTestNames return test names for TestSuite
func (t TestSuite) GetTestNames() []string {
	var names []string
	var steps []TestSuiteStep

	steps = append(steps, t.Before...)
	steps = append(steps, t.Steps...)
	steps = append(steps, t.After...)
	for _, step := range steps {
		if step.Execute != nil {
			names = append(names, step.Execute.Name)
		}
	}

	return set.Of(names...).ToArray()
}
