package grpc_test

import (
	"bytes"
	"encoding/json"
	"testing"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
)

func TestUpdateOrCreateTestWorkflow(t *testing.T) {
	var srv testSrv
	client := startGRPCTestConnection(t, &srv)

	input := testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Config: map[string]testworkflowsv1.ParameterSchema{
					"foo": {
						Description: "foo",
						Type:        "bar",
					},
					"baz": {
						Description: "baz",
						Type:        "qux",
					},
				},
				Concurrency: &testworkflowsv1.ConcurrencyPolicy{
					Group: "foo",
					Max:   5,
				},
			},
		},
	}
	expect, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}

	if err := client.UpdateOrCreateTestWorkflow(t.Context(), input); err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(expect, srv.TestWorkflow.GetPayload()) {
		t.Errorf("expect %v, got %v", expect, srv.TestWorkflow.GetPayload())
	}
}
