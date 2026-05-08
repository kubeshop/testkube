package controlplane

import (
	"context"
	"encoding/json"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowtemplateclient"
)

func (s *Server) WatchTestWorkflowTemplateUpdates(*cloud.WatchTestWorkflowTemplateUpdatesRequest, cloud.TestKubeCloudAPI_WatchTestWorkflowTemplateUpdatesServer) error {
	return status.Errorf(codes.Unimplemented, "method WatchTestWorkflowTemplateUpdates not implemented")
}

func (s *Server) GetTestWorkflowTemplate(ctx context.Context, req *cloud.GetTestWorkflowTemplateRequest) (*cloud.GetTestWorkflowTemplateResponse, error) {
	template, err := s.testWorkflowTemplatesClient.Get(ctx, "", req.Name)
	if err != nil {
		return nil, err
	}
	templateBytes, err := json.Marshal(template)
	if err != nil {
		return nil, err
	}
	return &cloud.GetTestWorkflowTemplateResponse{Template: templateBytes}, nil
}

func (s *Server) ListTestWorkflowTemplates(req *cloud.ListTestWorkflowTemplatesRequest, srv cloud.TestKubeCloudAPI_ListTestWorkflowTemplatesServer) error {
	templates, err := s.testWorkflowTemplatesClient.List(srv.Context(), "", testworkflowtemplateclient.ListOptions{
		Labels:     req.Labels,
		TextSearch: req.TextSearch,
		Offset:     req.Offset,
		Limit:      req.Limit,
	})
	if err != nil {
		return err
	}
	var templateBytes []byte
	for _, template := range templates {
		templateBytes, err = json.Marshal(template)
		if err != nil {
			return err
		}
		err = srv.Send(&cloud.TestWorkflowTemplateListItem{Template: templateBytes})
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) ListTestWorkflowTemplateLabels(ctx context.Context, req *cloud.ListTestWorkflowTemplateLabelsRequest) (*cloud.ListTestWorkflowTemplateLabelsResponse, error) {
	labels, err := s.testWorkflowTemplatesClient.ListLabels(ctx, "")
	if err != nil {
		return nil, err
	}
	res := &cloud.ListTestWorkflowTemplateLabelsResponse{Labels: make([]*cloud.LabelListItem, 0, len(labels))}
	for k, v := range labels {
		res.Labels = append(res.Labels, &cloud.LabelListItem{Name: k, Value: v})
	}
	return res, nil
}

func (s *Server) CreateTestWorkflowTemplate(ctx context.Context, req *cloud.CreateTestWorkflowTemplateRequest) (*cloud.CreateTestWorkflowTemplateResponse, error) {
	var template testkube.TestWorkflowTemplate
	err := json.Unmarshal(req.Template, &template)
	if err != nil {
		return nil, err
	}
	err = s.testWorkflowTemplatesClient.Create(ctx, "", template)
	if err != nil {
		return nil, err
	}
	return &cloud.CreateTestWorkflowTemplateResponse{}, nil
}

func (s *Server) UpdateTestWorkflowTemplate(ctx context.Context, req *cloud.UpdateTestWorkflowTemplateRequest) (*cloud.UpdateTestWorkflowTemplateResponse, error) {
	var template testkube.TestWorkflowTemplate
	err := json.Unmarshal(req.Template, &template)
	if err != nil {
		return nil, err
	}
	err = s.testWorkflowTemplatesClient.Update(ctx, "", template)
	if err != nil {
		return nil, err
	}
	return &cloud.UpdateTestWorkflowTemplateResponse{}, nil
}

func (s *Server) DeleteTestWorkflowTemplate(ctx context.Context, req *cloud.DeleteTestWorkflowTemplateRequest) (*cloud.DeleteTestWorkflowTemplateResponse, error) {
	err := s.testWorkflowTemplatesClient.Delete(ctx, "", req.Name)
	if err != nil {
		return nil, err
	}
	return &cloud.DeleteTestWorkflowTemplateResponse{}, nil
}

func (s *Server) DeleteTestWorkflowTemplatesByLabels(ctx context.Context, req *cloud.DeleteTestWorkflowTemplatesByLabelsRequest) (*cloud.DeleteTestWorkflowTemplatesByLabelsResponse, error) {
	count, err := s.testWorkflowTemplatesClient.DeleteByLabels(ctx, "", req.Labels)
	if err != nil {
		return nil, err
	}
	return &cloud.DeleteTestWorkflowTemplatesByLabelsResponse{Count: count}, nil
}
