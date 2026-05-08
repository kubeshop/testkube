package grpc_test

import (
	"bytes"
	"encoding/json"
	"testing"

	testtriggersv1 "github.com/kubeshop/testkube/api/testtriggers/v1"
)

func TestUpdateOrCreateTestTrigger(t *testing.T) {
	var srv testSrv
	client := startGRPCTestConnection(t, &srv)

	input := testtriggersv1.TestTrigger{
		Spec: testtriggersv1.TestTriggerSpec{
			Resource:  "foo",
			Event:     "bar",
			Action:    "baz",
			Execution: "qux",
		},
	}
	expect, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}

	if err := client.UpdateOrCreateTestTrigger(t.Context(), input); err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(expect, srv.TestTrigger.GetPayload()) {
		t.Errorf("expect %v, got %v", expect, srv.TestTrigger.GetPayload())
	}
}
