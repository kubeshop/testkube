package grpc_test

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/local"
	"google.golang.org/grpc/metadata"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	syncagent "github.com/kubeshop/testkube/internal/sync"
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

	// Namespace is the namespace of the last update.
	Namespace string
	// DeleteId and DeleteNamespace are the name and the namespace of the last delete.
	DeleteId        string
	DeleteNamespace string
	// EnvironmentID is the environment-id header of the last call.
	EnvironmentID string

	// Err is returned by both RPCs, to exercise the client's status code translation.
	Err error
}

// recordEnvironment keeps the environment-id header of the call, or empty when the call has none.
func (t *testSrv) recordEnvironment(ctx context.Context) {
	t.EnvironmentID = ""
	md, _ := metadata.FromIncomingContext(ctx)
	if values := md.Get("environment-id"); len(values) > 0 {
		t.EnvironmentID = values[0]
	}
}

func (t *testSrv) UpdateOrCreate(ctx context.Context, req *syncv1.UpdateOrCreateRequest) (*syncv1.UpdateOrCreateResponse, error) {
	t.recordEnvironment(ctx)
	t.Namespace = req.GetNamespace()
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

func (t *testSrv) Delete(ctx context.Context, req *syncv1.DeleteRequest) (*syncv1.DeleteResponse, error) {
	t.recordEnvironment(ctx)
	if id := req.GetTestWorkflow(); id != nil {
		t.DeleteId = id.GetId()
		t.DeleteNamespace = id.GetNamespace()
	}
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

	return syncgrpc.NewClient(conn, zap.NewExample().Sugar(), "foo", "bar", "env-default", true)
}

func TestClientTarget(t *testing.T) {
	tests := []struct {
		name   string
		target syncagent.Target
		delete bool
		check  func(t *testing.T, srv *testSrv)
	}{
		{
			name:   "the default Environment sends no environment-id header",
			target: syncagent.Target{Namespace: "team-a", EnvironmentID: "env-default"},
			check: func(t *testing.T, srv *testSrv) {
				assert.Empty(t, srv.EnvironmentID)
			},
		},
		{
			name:   "an empty Environment sends no header",
			target: syncagent.Target{Namespace: "team-a"},
			check: func(t *testing.T, srv *testSrv) {
				assert.Empty(t, srv.EnvironmentID)
			},
		},
		{
			name:   "another Environment sends the header",
			target: syncagent.Target{Namespace: "team-b", EnvironmentID: "env-b"},
			check: func(t *testing.T, srv *testSrv) {
				assert.Equal(t, "env-b", srv.EnvironmentID)
			},
		},
		{
			name:   "another Environment sends the header on a delete",
			target: syncagent.Target{Namespace: "team-b", EnvironmentID: "env-b"},
			delete: true,
			check: func(t *testing.T, srv *testSrv) {
				assert.Equal(t, "env-b", srv.EnvironmentID)
			},
		},
		{
			name:   "an update carries the namespace",
			target: syncagent.Target{Namespace: "team-a"},
			check: func(t *testing.T, srv *testSrv) {
				assert.Equal(t, "team-a", srv.Namespace)
			},
		},
		{
			name:   "a delete carries the namespace",
			target: syncagent.Target{Namespace: "team-a"},
			delete: true,
			check: func(t *testing.T, srv *testSrv) {
				assert.Equal(t, "smoke", srv.DeleteId)
				assert.Equal(t, "team-a", srv.DeleteNamespace)
			},
		},
		{
			name:   "the copy carries the source-namespace label and the Kubernetes object keeps its labels",
			target: syncagent.Target{Namespace: "team-a"},
			check: func(t *testing.T, srv *testSrv) {
				var sent testworkflowsv1.TestWorkflow
				require.NoError(t, json.Unmarshal(srv.TestWorkflow.GetJson(), &sent))
				assert.Equal(t, map[string]string{"app": "smoke", syncagent.SourceNamespaceLabel: "team-a"}, sent.Labels)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var srv testSrv
			client := startGRPCTestConnection(t, &srv)
			labels := map[string]string{"app": "smoke"}
			workflow := testworkflowsv1.TestWorkflow{ObjectMeta: metav1.ObjectMeta{Name: "smoke", Labels: labels}}

			if tt.delete {
				require.NoError(t, client.DeleteTestWorkflow(t.Context(), tt.target, "smoke"))
			} else {
				require.NoError(t, client.UpdateOrCreateTestWorkflow(t.Context(), tt.target, workflow))
			}

			tt.check(t, &srv)
			assert.Equal(t, map[string]string{"app": "smoke"}, labels, "the Kubernetes object must keep its labels")
		})
	}
}
