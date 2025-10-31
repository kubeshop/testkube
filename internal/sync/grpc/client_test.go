package grpc_test

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/local"

	syncgrpc "github.com/kubeshop/testkube/internal/sync/grpc"
	syncv1 "github.com/kubeshop/testkube/pkg/proto/testkube/sync/v1"
)

type testSrv struct {
	syncv1.UnimplementedSyncServiceServer

	TestTrigger          *syncv1.TestTrigger
	TestWorkflow         *syncv1.TestWorkflow
	TestWorkflowTemplate *syncv1.TestWorkflowTemplate
	Webhook              *syncv1.Webhook
	WebhookTemplate      *syncv1.WebhookTemplate
}

func (t *testSrv) UpdateOrCreate(_ context.Context, req *syncv1.UpdateOrCreateRequest) (*syncv1.UpdateOrCreateResponse, error) {
	switch v := req.Payload.(type) {
	case *syncv1.UpdateOrCreateRequest_TestTrigger:
		t.TestTrigger = v.TestTrigger
	case *syncv1.UpdateOrCreateRequest_TestWorkflow:
		t.TestWorkflow = v.TestWorkflow
	case *syncv1.UpdateOrCreateRequest_TestWorkflowTemplate:
		t.TestWorkflowTemplate = v.TestWorkflowTemplate
	case *syncv1.UpdateOrCreateRequest_Webhook:
		t.Webhook = v.Webhook
	case *syncv1.UpdateOrCreateRequest_WebhookTemplate:
		t.WebhookTemplate = v.WebhookTemplate
	}
	return nil, nil
}

func (t *testSrv) Delete(_ context.Context, req *syncv1.DeleteRequest) (*syncv1.DeleteResponse, error) {
	return nil, nil
}

func startGRPCTestConnection(t *testing.T, ts *testSrv) syncgrpc.Client {
	t.Helper()

	srv := grpc.NewServer(grpc.Creds(local.NewCredentials()))

	syncv1.RegisterSyncServiceServer(srv, ts)

	socketAddr := filepath.Join(os.TempDir(), t.Name()+".sock")
	t.Cleanup(func() {
		os.Remove(socketAddr)
	})

	listener, err := net.Listen("unix", socketAddr)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		if err := srv.Serve(listener); err != nil {
			t.Error(err)
			return
		}
	}()

	t.Cleanup(srv.Stop)

	// Connecting over a unix socket requires three slashes.
	// - Two for the schema (standard).
	// - One after the "authority", which for UDS doesnt exist.
	conn, err := grpc.NewClient("unix:///"+socketAddr, grpc.WithTransportCredentials(local.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}

	return syncgrpc.NewClient(conn, zap.NewExample().Sugar(), "foo", "bar")
}
