package testworkflowclient

import (
	"context"
	"encoding/json"
	"errors"
	"math"

	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	cloudtestworkflow "github.com/kubeshop/testkube/pkg/cloud/data/testworkflow"
)

var _ TestWorkflowClient = &cloudTestWorkflowClient{}

type cloudTestWorkflowClient struct {
	conn   *grpc.ClientConn
	client cloud.TestKubeCloudAPIClient
	apiKey string
}

func NewCloudTestWorkflowClient(conn *grpc.ClientConn, apiKey string) TestWorkflowClient {
	return &cloudTestWorkflowClient{
		conn:   conn,
		client: cloud.NewTestKubeCloudAPIClient(conn),
		apiKey: apiKey,
	}
}

// TODO: Prepare separate Control Plane function for that
func (c *cloudTestWorkflowClient) Get(ctx context.Context, environmentId string, name string) (*testkube.TestWorkflow, error) {
	// Pass the additional information
	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("api-key", c.apiKey, "environment-id", environmentId))

	// Build the request
	jsonPayload, err := json.Marshal(cloudtestworkflow.TestWorkflowGetRequest{Name: name})
	if err != nil {
		return nil, err
	}
	s := structpb.Struct{}
	if err := s.UnmarshalJSON(jsonPayload); err != nil {
		return nil, err
	}
	req := cloud.CommandRequest{
		Command: string(cloudtestworkflow.CmdTestWorkflowGet),
		Payload: &s,
	}

	// Call the gRPC API
	opts := []grpc.CallOption{grpc.UseCompressor(gzip.Name), grpc.MaxCallRecvMsgSize(math.MaxInt32)}
	cmdResponse, err := c.client.Call(ctx, &req, opts...)
	if err != nil {
		return nil, err
	}

	// Retrieve the response
	var commandResponse cloudtestworkflow.TestWorkflowGetResponse
	if err := json.Unmarshal(cmdResponse.Response, &commandResponse); err != nil {
		return nil, err
	}
	return &commandResponse.TestWorkflow, nil
}

// TODO:
func (c *cloudTestWorkflowClient) List(ctx context.Context, environmentId string, labels map[string]string) ([]testkube.TestWorkflow, error) {
	return nil, errors.New("not implemented yet")
}

// TODO:
func (c *cloudTestWorkflowClient) ListLabels(ctx context.Context, environmentId string) (map[string][]string, error) {
	return nil, errors.New("not implemented yet")
}

// TODO:
func (c *cloudTestWorkflowClient) Update(ctx context.Context, environmentId string, workflow testkube.TestWorkflow) error {
	return errors.New("not implemented yet")
}

// TODO:
func (c *cloudTestWorkflowClient) Create(ctx context.Context, environmentId string, workflow testkube.TestWorkflow) error {
	return errors.New("not implemented yet")
}

// TODO:
func (c *cloudTestWorkflowClient) Delete(ctx context.Context, environmentId string, name string) error {
	return errors.New("not implemented yet")
}

// TODO:
func (c *cloudTestWorkflowClient) DeleteByLabels(ctx context.Context, environmentId string, labels map[string]string) error {
	return errors.New("not implemented yet")
}
