package grpc_test

import (
	"errors"
	"strings"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	executorv1 "github.com/kubeshop/testkube/api/executor/v1"
	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	syncagent "github.com/kubeshop/testkube/internal/sync"
	syncgrpc "github.com/kubeshop/testkube/internal/sync/grpc"
)

func TestTranslateError(t *testing.T) {
	updateWorkflow := func(t *testing.T, c syncgrpc.Client) error {
		return c.UpdateOrCreateTestWorkflow(t.Context(), testworkflowsv1.TestWorkflow{})
	}
	deleteWorkflow := func(t *testing.T, c syncgrpc.Client) error {
		return c.DeleteTestWorkflow(t.Context(), "smoke")
	}
	updateWebhook := func(t *testing.T, c syncgrpc.Client) error {
		return c.UpdateOrCreateWebhook(t.Context(), executorv1.Webhook{})
	}

	tests := []struct {
		name string
		code codes.Code
		call func(*testing.T, syncgrpc.Client) error
		// want is the sentinel that the error wraps, or nil for an error that stays retryable.
		want error
	}{
		{name: "FailedPrecondition on update is an ownership conflict", code: codes.FailedPrecondition, call: updateWorkflow, want: syncagent.ErrOwnershipConflict},
		{name: "FailedPrecondition on delete is an ownership conflict", code: codes.FailedPrecondition, call: deleteWorkflow, want: syncagent.ErrOwnershipConflict},
		{name: "InvalidArgument is an invalid resource", code: codes.InvalidArgument, call: updateWebhook, want: syncagent.ErrInvalidResource},
		{name: "Internal stays retryable", code: codes.Internal, call: updateWorkflow},
		{name: "Unavailable stays retryable", code: codes.Unavailable, call: updateWorkflow},
		{name: "NotFound stays retryable", code: codes.NotFound, call: updateWorkflow},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			const message = "message of the Control Plane"
			client := startGRPCTestConnection(t, &testSrv{Err: status.Error(tt.code, message)})

			err := tt.call(t, client)

			if err == nil {
				t.Fatal("expected an error")
			}
			if tt.want == nil {
				if syncagent.IsRejection(err) {
					t.Errorf("expected a retryable error, got rejection %v", err)
				}
				return
			}
			if !errors.Is(err, tt.want) {
				t.Errorf("expected %v in the chain, got %v", tt.want, err)
			}
			if !strings.Contains(err.Error(), message) {
				t.Errorf("expected the message of the Control Plane to stay in the error, because it names the owner or the invalid value, got %v", err)
			}
		})
	}
}
