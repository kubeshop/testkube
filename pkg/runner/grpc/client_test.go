package grpc_test

import (
	"context"
	"crypto/x509"
	"net"
	"testing"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	executionv1 "github.com/kubeshop/testkube/pkg/proto/testkube/testworkflow/execution/v1"
	runnergrpc "github.com/kubeshop/testkube/pkg/runner/grpc"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
)

func TestClient_IsSupported(t *testing.T) {
	// Set up TLS
	ca, err := x509.SystemCertPool()
	if err != nil {
		t.Fatal(err)
	}
	caCert, cert := generateCertificate(t)
	ca.AddCert(caCert)

	// Start server.
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := grpc.NewServer(
		grpc.Creds(credentials.NewServerTLSFromCert(cert)),
	)
	executionv1.RegisterTestWorkflowExecutionServiceServer(srv, &executionv1.UnimplementedTestWorkflowExecutionServiceServer{})
	go func() {
		if err := srv.Serve(listener); err != nil {
			t.Log(err)
		}
	}()

	// Connect to server.
	conn, err := grpc.NewClient(listener.Addr().String(),
		grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(ca, "")),
	)
	if err != nil {
		t.Fatal(err)
	}

	client := runnergrpc.NewClient(
		conn,
		zap.NewExample().Sugar(),
		testRunner{},
		"foo",
		"bar",
		testworkflowconfig.ControlPlaneConfig{},
		testWorkflowStore{},
	)

	supported := client.IsSupported(t.Context(), "baz")
	if !supported {
		// Success!
		return
	}

	t.Errorf("Inccorectly returned supported when is not.")
}

type testRunner struct{}

func (t testRunner) Execute(request executionworkertypes.ExecuteRequest) (*executionworkertypes.ExecuteResult, error) {
	//TODO implement me
	panic("implement me")
}

func (t testRunner) Pause(executionId string) error {
	//TODO implement me
	panic("implement me")
}

func (t testRunner) Resume(executionId string) error {
	//TODO implement me
	panic("implement me")
}

func (t testRunner) Abort(executionId string) error {
	//TODO implement me
	panic("implement me")
}

func (t testRunner) Cancel(executionId string) error {
	//TODO implement me
	panic("implement me")
}

type testWorkflowStore struct{}

func (t testWorkflowStore) Get(ctx context.Context, environmentId string, name string) (*testkube.TestWorkflow, error) {
	//TODO implement me
	panic("implement me")
}
