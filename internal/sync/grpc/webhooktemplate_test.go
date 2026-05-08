package grpc_test

import (
	"bytes"
	"encoding/json"
	"testing"

	executorv1 "github.com/kubeshop/testkube/api/executor/v1"
)

func TestUpdateOrCreateWebhookTemplate(t *testing.T) {
	var srv testSrv
	client := startGRPCTestConnection(t, &srv)

	input := executorv1.WebhookTemplate{
		Spec: executorv1.WebhookTemplateSpec{
			Uri:                "foo",
			Selector:           "bar",
			PayloadObjectField: "baz",
			PayloadTemplate:    "qux",
		},
	}
	expect, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}

	if err := client.UpdateOrCreateWebhookTemplate(t.Context(), input); err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(expect, srv.WebhookTemplate.GetPayload()) {
		t.Errorf("expect %v, got %v", expect, srv.WebhookTemplate.GetPayload())
	}
}
