package grpc_test

import (
	"errors"
	"strings"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	syncagent "github.com/kubeshop/testkube/internal/sync"
)

// The Control Plane signals an ownership conflict with FailedPrecondition. Callers act on the
// sentinel rather than on gRPC codes, so the client has to translate it.
func TestOwnershipConflictIsRecognised(t *testing.T) {
	const rejection = `TestWorkflow "smoke" is owned by GitOps agent "team-a-gitops"`
	srv := testSrv{Err: status.Error(codes.FailedPrecondition, rejection)}
	client := startGRPCTestConnection(t, &srv)

	tests := map[string]func() error{
		"update": func() error {
			return client.UpdateOrCreateTestWorkflow(t.Context(), testworkflowsv1.TestWorkflow{})
		},
		"delete": func() error {
			return client.DeleteTestWorkflow(t.Context(), "smoke")
		},
	}

	for name, call := range tests {
		t.Run(name, func(t *testing.T) {
			err := call()

			if !errors.Is(err, syncagent.ErrOwnershipConflict) {
				t.Errorf("expected an ownership conflict, got %v", err)
			}
			// The Control Plane's message names the owner, so it has to survive translation.
			if !strings.Contains(err.Error(), rejection) {
				t.Errorf("expected the Control Plane's message to be preserved, got %v", err)
			}
		})
	}
}

// Any other code is a transient or unrelated failure and must not be mistaken for a conflict, so
// that the reconciler keeps retrying it.
func TestOtherStatusCodesAreNotConflicts(t *testing.T) {
	for _, code := range []codes.Code{codes.Internal, codes.Unavailable, codes.InvalidArgument, codes.NotFound} {
		t.Run(code.String(), func(t *testing.T) {
			srv := testSrv{Err: status.Error(code, "boom")}
			client := startGRPCTestConnection(t, &srv)

			err := client.UpdateOrCreateTestWorkflow(t.Context(), testworkflowsv1.TestWorkflow{})

			if err == nil {
				t.Fatal("expected an error")
			}
			if errors.Is(err, syncagent.ErrOwnershipConflict) {
				t.Errorf("%s must not be treated as an ownership conflict, got %v", code, err)
			}
		})
	}
}
