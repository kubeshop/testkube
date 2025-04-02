package testkube

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTestWorkflowEqualsUnorderedMap(t *testing.T) {
	o := &TestWorkflow{
		Spec: &TestWorkflowSpec{
			Config: map[string]TestWorkflowParameterSchema{
				"name1":  {Description: "some name"},
				"name2":  {Description: "another name"},
				"name3":  {Description: "another name"},
				"name6":  {Description: "another name"},
				"name7":  {Description: "another name"},
				"name4":  {Description: "another name"},
				"name5":  {Description: "another name"},
				"name8":  {Description: "another name"},
				"name9":  {Description: "another name"},
				"name10": {Description: "another name"},
				"name11": {Description: "another name"},
				"name12": {Description: "another name"},
				"name13": {Description: "another name"},
			},
		},
	}
	assert.Equal(t, true, o.Equals(o.DeepCopy()))
}
