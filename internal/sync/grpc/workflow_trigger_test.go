package grpc_test

import (
	"bytes"
	"encoding/json"
	"testing"

	workflowtriggersv1 "github.com/kubeshop/testkube/api/workflowtriggers/v1"
)

func TestUpdateOrCreateWorkflowTrigger(t *testing.T) {
	var srv testSrv
	client := startGRPCTestConnection(t, &srv)

	input := workflowtriggersv1.WorkflowTrigger{
		Spec: workflowtriggersv1.WorkflowTriggerSpec{
			When: workflowtriggersv1.WorkflowTriggerWhen{Event: "modified"},
			Run: workflowtriggersv1.WorkflowTriggerRun{
				Workflow: workflowtriggersv1.WorkflowTriggerWorkflowSelector{Name: "smoke-test"},
			},
		},
	}
	expect, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}

	if err := client.UpdateOrCreateWorkflowTrigger(t.Context(), input); err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(expect, srv.WorkflowTrigger.GetPayload()) {
		t.Errorf("expect %v, got %v", expect, srv.WorkflowTrigger.GetPayload())
	}
}
