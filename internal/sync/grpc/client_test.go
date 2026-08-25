package grpc_test

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/local"

	syncgrpc "github.com/kubeshop/testkube/internal/sync/grpc"
	syncv1 "github.com/kubeshop/testkube/pkg/proto/testkube/sync/v1"
	testworkflowv1 "github.com/kubeshop/testkube/pkg/proto/testkube/testworkflow/v1"
)

type testSrv struct {
	syncv1.UnimplementedSyncServiceServer

	TestTrigger          *syncv1.TestTrigger
	WorkflowTrigger      *syncv1.WorkflowTrigger
	TestWorkflow         *testworkflowv1.TestWorkflow
	TestWorkflowTemplate *syncv1.TestWorkflowTemplate
	Webhook              *syncv1.Webhook
	WebhookTemplate      *syncv1.WebhookTemplate

	// Err is returned by both RPCs, to exercise the client's status code translation.
	Err error
}

func (t *testSrv) UpdateOrCreate(_ context.Context, req *syncv1.UpdateOrCreateRequest) (*syncv1.UpdateOrCreateResponse, error) {
	switch v := req.Payload.(type) {
	case *syncv1.UpdateOrCreateRequest_TestTrigger:
		t.TestTrigger = v.TestTrigger
	case *syncv1.UpdateOrCreateRequest_WorkflowTrigger:
		t.WorkflowTrigger = v.WorkflowTrigger
	case *syncv1.UpdateOrCreateRequest_TestWorkflow:
		t.TestWorkflow = v.TestWorkflow
	case *syncv1.UpdateOrCreateRequest_TestWorkflowTemplate:
		t.TestWorkflowTemplate = v.TestWorkflowTemplate
	case *syncv1.UpdateOrCreateRequest_Webhook:
		t.Webhook = v.Webhook
	case *syncv1.UpdateOrCreateRequest_WebhookTemplate:
		t.WebhookTemplate = v.WebhookTemplate
	}
	return nil, t.Err
}

func (t *testSrv) Delete(_ context.Context, req *syncv1.DeleteRequest) (*syncv1.DeleteResponse, error) {
	return nil, t.Err
}

// socketPath derives a listenable socket path from the test name. Slashes from subtest names would
// be read as directories, and the whole path has to stay inside the ~104 byte limit that macOS
// imposes on unix socket addresses, so long names are trimmed from the front where they are least
// distinctive.
func socketPath(t *testing.T) string {
	t.Helper()

	dir := os.TempDir()
	name := strings.ReplaceAll(t.Name(), "/", "_")

	const maxPathLen = 100
	if budget := maxPathLen - len(dir) - len("/.sock"); budget > 0 && len(name) > budget {
		name = name[len(name)-budget:]
	}

	return filepath.Join(dir, name+".sock")
}

func startGRPCTestConnection(t *testing.T, ts *testSrv) syncgrpc.Client {
	t.Helper()

	srv := grpc.NewServer(grpc.Creds(local.NewCredentials()))

	syncv1.RegisterSyncServiceServer(srv, ts)

	socketAddr := socketPath(t)
	t.Cleanup(func() {
		os.Remove(socketAddr)
	})

	var lc net.ListenConfig
	listener, err := lc.Listen(context.Background(), "unix", socketAddr)
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

	return syncgrpc.NewClient(conn, zap.NewExample().Sugar(), "foo", "bar", true)
}
