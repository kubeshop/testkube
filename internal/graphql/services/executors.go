package services

import (
	"context"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	executorsmapper "github.com/kubeshop/testkube/pkg/mapper/executors"
	executorsclientv1 "github.com/kubeshop/testkube/pkg/operator/client/executors/v1"
)

//go:generate mockgen -destination=./mock_executors.go -package=services "github.com/kubeshop/testkube/internal/graphql/services" ExecutorsService
type ExecutorsService interface {
	List(selector string) ([]testkube.ExecutorDetails, error)
	SubscribeList(ctx context.Context, selector string) (<-chan []testkube.ExecutorDetails, error)
}

type executorsService struct {
	ServiceBase
	client executorsclientv1.Interface
}

func NewExecutorsService(service Service, client executorsclientv1.Interface) ExecutorsService {
	return &executorsService{ServiceBase: ServiceBase{Service: service}, client: client}
}

func (s *executorsService) List(selector string) ([]testkube.ExecutorDetails, error) {
	execs, err := s.client.List(selector)
	if err != nil {
		return nil, err
	}
	return Map(execs.Items, executorsmapper.MapExecutorCRDToExecutorDetails), nil
}

func (s *executorsService) SubscribeList(ctx context.Context, selector string) (<-chan []testkube.ExecutorDetails, error) {
	return HandleSubscription(ctx, "agentevents.executor.>", s, func() ([]testkube.ExecutorDetails, error) {
		return s.List(selector)
	})
}
