package testkube

import (
	"fmt"
)

type TestSuites []TestSuite

func (tests TestSuites) Table() (header []string, output [][]string) {
	header = []string{"Name", "Description", "Steps", "Labels", "Schedule"}
	for _, e := range tests {
		output = append(output, []string{
			e.Name,
			e.Description,
			fmt.Sprintf("%d", len(e.Steps)),
			LabelsToString(e.Labels),
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
