package testkube

import (
	"encoding/json"
	"fmt"
)

type Tests []Test

func (tests Tests) Table() (header []string, output [][]string) {
	header = []string{"Name", "Description", "Steps"}
	for _, e := range tests {
		output = append(output, []string{
			e.Name,
			e.Description,
			fmt.Sprintf("%d", len(e.Steps)),
		})
	}

	return
}

// TODO rethink this struct and handling as for now we want to simplify OpenAPI spec
// and this one need to be manually synchronized with genered files on change!
// TestBase Intermidiate struct to handle convertion from interface to structs for steps
type TestBase struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	// Run this step before whole suite
	Before []map[string]interface{} `json:"before,omitempty"`
	// Steps to run
	Steps []map[string]interface{} `json:"steps"`
	// Run this step after whole suite
	After   []map[string]interface{} `json:"after,omitempty"`
	Repeats int32                    `json:"repeats,omitempty"`
}

func (test *Test) UnmarshalJSON(data []byte) error {
	var t TestBase
	err := json.Unmarshal(data, &t)
	if err != nil {
		return err
	}

	test.Name = t.Name
	test.Description = t.Description
	test.Repeats = t.Repeats

	for _, step := range t.Steps {
		if s := TestStepBase(step).GetTestStep(); s != nil {
			test.Steps = append(test.Steps, s)
		}
	}

	return nil
}
