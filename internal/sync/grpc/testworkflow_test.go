package grpc_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

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

	if !bytes.Equal(expect, srv.TestWorkflow.GetJson()) {
		t.Errorf("expect %v, got %v", expect, srv.TestWorkflow.GetJson())
	}
}

func TestDeleteTestWorkflow(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		want      *string
	}{
		{name: "the namespace is sent", namespace: "team-a", want: proto.String("team-a")},
		{name: "an empty namespace is left unset", namespace: "", want: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var srv testSrv
			client := startGRPCTestConnection(t, &srv)

			require.NoError(t, client.DeleteTestWorkflow(t.Context(), "foobar", tt.namespace))

			id := srv.DeleteRequest.GetTestWorkflow()
			assert.Equal(t, "foobar", id.GetId())
			assert.Equal(t, tt.want, id.Namespace)
		})
	}
}
