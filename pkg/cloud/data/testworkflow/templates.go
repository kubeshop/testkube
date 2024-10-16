package testworkflow

import (
	"context"
	"encoding/json"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	testworkflowsclientv1 "github.com/kubeshop/testkube-operator/pkg/client/testworkflows/v1"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/cloud/data/executor"
	testworkflowmappers "github.com/kubeshop/testkube/pkg/mapper/testworkflows"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

var _ testworkflowsclientv1.TestWorkflowTemplatesInterface = (*CloudTestWorkflowTemplateRepository)(nil)

type CloudTestWorkflowTemplateRepository struct {
	executor executor.Executor
}

func NewCloudTestWorkflowTemplateRepository(client cloud.TestKubeCloudAPIClient, grpcConn *grpc.ClientConn, apiKey string) *CloudTestWorkflowTemplateRepository {
	return &CloudTestWorkflowTemplateRepository{executor: executor.NewCloudGRPCExecutor(client, grpcConn, apiKey)}
}

func (r *CloudTestWorkflowTemplateRepository) List(selector string) (*testworkflowsv1.TestWorkflowTemplateList, error) {
	return nil, errors.New("unimplemented")
}

func (r *CloudTestWorkflowTemplateRepository) ListLabels() (map[string][]string, error) {
	return make(map[string][]string), nil
}

func (r *CloudTestWorkflowTemplateRepository) Get(name string) (*testworkflowsv1.TestWorkflowTemplate, error) {
	req := TestWorkflowTemplateGetRequest{Name: name}
	response, err := r.executor.Execute(context.Background(), CmdTestWorkflowTemplateGet, req)
	if err != nil {
		return nil, err
	}
	var commandResponse TestWorkflowTemplateGetResponse
	if err := json.Unmarshal(response, &commandResponse); err != nil {
		return nil, err
	}
	return testworkflowmappers.MapTemplateAPIToKube(&commandResponse.TestWorkflowTemplate), nil
}

// Create creates new TestWorkflow
func (r *CloudTestWorkflowTemplateRepository) Create(workflow *testworkflowsv1.TestWorkflowTemplate) (*testworkflowsv1.TestWorkflowTemplate, error) {
	return nil, errors.New("unimplemented")
}

func (r *CloudTestWorkflowTemplateRepository) Update(workflow *testworkflowsv1.TestWorkflowTemplate) (*testworkflowsv1.TestWorkflowTemplate, error) {
	return nil, errors.New("unimplemented")
}

func (r *CloudTestWorkflowTemplateRepository) Apply(workflow *testworkflowsv1.TestWorkflowTemplate) error {
	return errors.New("unimplemented")
}

func (r *CloudTestWorkflowTemplateRepository) Delete(name string) error {
	return errors.New("unimplemented")
}

func (r *CloudTestWorkflowTemplateRepository) DeleteAll() error {
	return errors.New("unimplemented")
}

func (r *CloudTestWorkflowTemplateRepository) DeleteByLabels(selector string) error {
	return errors.New("unimplemented")
}

func (r *CloudTestWorkflowTemplateRepository) UpdateStatus(workflow *testworkflowsv1.TestWorkflowTemplate) error {
	// This is the actual implementation, as update status
	// should update k8s crd's status field, but we don't have it when stored in mongo
	return nil
}
