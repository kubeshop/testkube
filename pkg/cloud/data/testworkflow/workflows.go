package testworkflow

import (
	"context"
	"encoding/json"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	testworkflowsclientv1 "github.com/kubeshop/testkube-operator/pkg/client/testworkflows/v1"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/cloud/data/executor"
	testworkflowmappers "github.com/kubeshop/testkube/pkg/mapper/testworkflows"

	"google.golang.org/grpc"
)

var _ testworkflowsclientv1.Interface = (*CloudTestWorkflowRepository)(nil)

type CloudTestWorkflowRepository struct {
	executor executor.Executor
}

func NewCloudTestWorkflowRepository(client cloud.TestKubeCloudAPIClient, grpcConn *grpc.ClientConn, apiKey string) *CloudTestWorkflowRepository {
	return &CloudTestWorkflowRepository{executor: executor.NewCloudGRPCExecutor(client, grpcConn, apiKey)}
}

func (r *CloudTestWorkflowRepository) List(selector string) (*testworkflowsv1.TestWorkflowList, error) {
	req := TestWorkflowListRequest{Selector: selector}
	response, err := r.executor.Execute(context.Background(), CmdTestWorkflowList, req)
	if err != nil {
		return nil, err
	}
	var commandResponse TestWorkflowListResponse
	if err := json.Unmarshal(response, &commandResponse); err != nil {
		return nil, err
	}
	list := testworkflowmappers.MapListAPIToKube(commandResponse.TestWorkflows)
	return &list, nil
}

func (r *CloudTestWorkflowRepository) ListLabels() (map[string][]string, error) {
	return make(map[string][]string), nil
}

func (r *CloudTestWorkflowRepository) Get(name string) (*testworkflowsv1.TestWorkflow, error) {
	req := TestWorkflowGetRequest{Name: name}
	response, err := r.executor.Execute(context.Background(), CmdTestWorkflowGet, req)
	if err != nil {
		return nil, err
	}
	var commandResponse TestWorkflowGetResponse
	if err := json.Unmarshal(response, &commandResponse); err != nil {
		return nil, err
	}
	return testworkflowmappers.MapAPIToKube(&commandResponse.TestWorkflow), nil
}

// Create creates new TestWorkflow
func (r *CloudTestWorkflowRepository) Create(workflow *testworkflowsv1.TestWorkflow) (*testworkflowsv1.TestWorkflow, error) {
	return nil, nil
}

func (r *CloudTestWorkflowRepository) Update(workflow *testworkflowsv1.TestWorkflow) (*testworkflowsv1.TestWorkflow, error) {
	return nil, nil
}

func (r *CloudTestWorkflowRepository) Apply(workflow *testworkflowsv1.TestWorkflow) error {
	return nil
}

func (r *CloudTestWorkflowRepository) Delete(name string) error {
	return nil
}

func (r *CloudTestWorkflowRepository) DeleteAll() error {
	return nil
}

func (r *CloudTestWorkflowRepository) DeleteByLabels(selector string) error {
	return nil
}

func (r *CloudTestWorkflowRepository) UpdateStatus(workflow *testworkflowsv1.TestWorkflow) error {
	return nil
}
