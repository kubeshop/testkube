package testworkflowexecutor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"math"
	"slices"
	"time"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/metadata"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/log"
	testworkflows2 "github.com/kubeshop/testkube/pkg/mapper/testworkflows"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowclient"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowtemplateclient"
	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
	"github.com/kubeshop/testkube/pkg/testworkflows"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowresolver"
)

const (
	SaveResultRetryMaxAttempts = 100
	SaveResultRetryBaseDelay   = 300 * time.Millisecond
)

//go:generate mockgen -destination=./mock_scheduler.go -package=testworkflowexecutor "github.com/kubeshop/testkube/pkg/testworkflows/testworkflowexecutor" Scheduler
type Scheduler interface {
	Schedule(ctx context.Context, sensitiveDataHandler SensitiveDataHandler, environmentId string, req *cloud.ScheduleRequest) (<-chan *testkube.TestWorkflowExecution, error)
	CriticalError(execution *testkube.TestWorkflowExecution, name string, err error) error
	Start(execution *testkube.TestWorkflowExecution) error
}

type scheduler struct {
	logger                      *zap.SugaredLogger
	testWorkflowsClient         testworkflowclient.TestWorkflowClient
	testWorkflowTemplatesClient testworkflowtemplateclient.TestWorkflowTemplateClient
	resultsRepository           testworkflow.Repository
	outputRepository            testworkflow.OutputRepository
	getRunners                  func(environmentId string, target *cloud.ExecutionTarget) ([]map[string]string, error)
	globalTemplateName          string
	organizationId              string
	defaultEnvironmentId        string

	agentId              string
	grpcClient           cloud.TestKubeCloudAPIClient
	grpcApiToken         string
	newExecutionsEnabled bool
}

func NewScheduler(
	testWorkflowsClient testworkflowclient.TestWorkflowClient,
	testWorkflowTemplatesClient testworkflowtemplateclient.TestWorkflowTemplateClient,
	resultsRepository testworkflow.Repository,
	outputRepository testworkflow.OutputRepository,
	getRunners func(environmentId string, target *cloud.ExecutionTarget) ([]map[string]string, error),
	globalTemplateName string,
	organizationId string,
	defaultEnvironmentId string,

	agentId string,
	grpcClient cloud.TestKubeCloudAPIClient,
	grpcApiToken string,
	newExecutionsEnabled bool,
) Scheduler {
	return &scheduler{
		logger:                      log.DefaultLogger,
		testWorkflowsClient:         testWorkflowsClient,
		testWorkflowTemplatesClient: testWorkflowTemplatesClient,
		resultsRepository:           resultsRepository,
		outputRepository:            outputRepository,
		getRunners:                  getRunners,
		globalTemplateName:          globalTemplateName,
		organizationId:              organizationId,
		defaultEnvironmentId:        defaultEnvironmentId,

		agentId:              agentId,
		grpcClient:           grpcClient,
		grpcApiToken:         grpcApiToken,
		newExecutionsEnabled: newExecutionsEnabled,
	}
}

func (s *scheduler) insert(ctx context.Context, execution *testkube.TestWorkflowExecution) error {
	err := retry(SaveResultRetryMaxAttempts, SaveResultRetryBaseDelay, func() error {
		err := s.resultsRepository.Insert(ctx, *execution)
		if err != nil {
			s.logger.Warnw("failed to update the TestWorkflow execution in database", "recoverable", true, "executionId", execution.Id, "error", err)
		}
		return err
	})
	if err != nil {
		s.logger.Errorw("failed to update the TestWorkflow execution in database", "recoverable", false, "executionId", execution.Id, "error", err)
	}
	return err
}

func (s *scheduler) update(ctx context.Context, execution *testkube.TestWorkflowExecution) error {
	err := retry(SaveResultRetryMaxAttempts, SaveResultRetryBaseDelay, func() error {
		err := s.resultsRepository.Update(ctx, *execution)
		if err != nil {
			s.logger.Warnw("failed to update the TestWorkflow execution in database", "recoverable", true, "executionId", execution.Id, "error", err)
		}
		return err
	})
	if err != nil {
		s.logger.Errorw("failed to update the TestWorkflow execution in database", "recoverable", false, "executionId", execution.Id, "error", err)
	}
	return err
}

func (s *scheduler) init(ctx context.Context, execution *testkube.TestWorkflowExecution) error {
	err := retry(SaveResultRetryMaxAttempts, SaveResultRetryBaseDelay, func() error {
		err := s.resultsRepository.Init(ctx, execution.Id, testworkflow.InitData{
			RunnerID:  execution.RunnerId,
			Namespace: execution.Namespace,
			Signature: execution.Signature,
		})
		if err != nil {
			s.logger.Warnw("failed to initialize the TestWorkflow execution in database", "recoverable", true, "executionId", execution.Id, "error", err)
		}
		return err
	})
	if err != nil {
		s.logger.Errorw("failed to initialize the TestWorkflow execution in database", "recoverable", false, "executionId", execution.Id, "error", err)
	}
	return err
}

func (s *scheduler) saveEmptyLogs(ctx context.Context, execution *testkube.TestWorkflowExecution) error {
	err := retry(SaveResultRetryMaxAttempts, SaveResultRetryBaseDelay, func() error {
		return s.outputRepository.SaveLog(ctx, execution.Id, execution.Workflow.Name, bytes.NewReader(nil))
	})
	if err != nil {
		s.logger.Errorw("failed to save empty log", "executionId", execution.Id, "error", err)
	}
	return err
}

func (s *scheduler) Schedule(ctx context.Context, sensitiveDataHandler SensitiveDataHandler, environmentId string, req *cloud.ScheduleRequest) (<-chan *testkube.TestWorkflowExecution, error) {
	// Prepare the channel
	ch := make(chan *testkube.TestWorkflowExecution, 1)

	// Set up context
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Ensure environment ID
	if environmentId == "" {
		environmentId = s.defaultEnvironmentId
	}

	// Validate the execution request
	if err := ValidateExecutionRequest(req); err != nil {
		close(ch)
		return ch, err
	}

	// Check if there is anything to run
	if len(req.Executions) == 0 {
		close(ch)
		return ch, nil
	}

	// Initialize execution template
	now := time.Now().UTC()
	base := NewIntermediateExecution().
		SetGroupID(primitive.NewObjectIDFromTimestamp(now).Hex()).
		SetScheduledAt(now).
		AppendTags(req.Tags).
		SetDisabledWebhooks(req.DisableWebhooks).
		SetKubernetesObjectName(req.KubernetesObjectName).
		SetRunningContext(GetLegacyRunningContext(req)).
		PrependTemplate(s.globalTemplateName)

	// Initialize fetchers
	testWorkflows := NewTestWorkflowFetcher(s.testWorkflowsClient, environmentId)
	testWorkflowTemplates := NewTestWorkflowTemplateFetcher(s.testWorkflowTemplatesClient, environmentId)

	// Prefetch all the Test Workflows
	err := testWorkflows.PrefetchMany(common.MapSlice(req.Executions, func(t *cloud.ScheduleExecution) *cloud.ScheduleResourceSelector {
		return t.Selector
	}))
	if err != nil {
		close(ch)
		return ch, err
	}

	// Prefetch all the Test Workflow Templates.
	// Don't fail immediately - it should be execution's error message if it's missing.
	tplNames := testWorkflows.TemplateNames()
	if s.globalTemplateName != "" {
		tplNames[testworkflowresolver.GetInternalTemplateName(s.globalTemplateName)] = struct{}{}
	}
	_ = testWorkflowTemplates.PrefetchMany(tplNames)

	// Flatten selectors
	intermediateSelectors := make([]*cloud.ScheduleExecution, 0, len(req.Executions))
	for _, execution := range req.Executions {
		list, _ := testWorkflows.Get(execution.Selector)
		for _, w := range list {
			intermediateSelectors = append(intermediateSelectors, &cloud.ScheduleExecution{
				Selector:      &cloud.ScheduleResourceSelector{Name: w.Name},
				Targets:       execution.Targets,
				Config:        execution.Config,
				ExecutionName: execution.ExecutionName, // TODO: what to do when execution name is configured, but multiple requested?
				Tags:          execution.Tags,
			})
		}
	}

	// Flatten target replicas
	originalTargets := make([]*cloud.ExecutionTarget, 0, len(intermediateSelectors))
	selectors := make([]*cloud.ScheduleExecution, 0, len(intermediateSelectors))
	for _, execution := range intermediateSelectors {
		// Ignore when no specific targets are passed
		if len(execution.Targets) == 0 {
			selectors = append(selectors, execution)
			originalTargets = append(originalTargets, &cloud.ExecutionTarget{})
			continue
		}

		for _, target := range execution.Targets {
			// Optimize repeating target - if there is filter on label, avoid doing the repeat
			replicate := make([]string, 0)
			for i := 0; i < len(target.Replicate); i++ {
				if _, ok := target.Match[target.Replicate[i]]; ok {
					continue
				}
				replicate = append(replicate, target.Replicate[i])
			}

			// Do not replicate if it's not expected
			if len(replicate) == 0 {
				selectors = append(selectors, &cloud.ScheduleExecution{
					Selector:      execution.Selector,
					Targets:       []*cloud.ExecutionTarget{{Match: target.Match, Not: target.Not}},
					Config:        execution.Config,
					ExecutionName: execution.ExecutionName, // TODO: what to do when execution name is configured, but multiple requested?
					Tags:          execution.Tags,
				})
				originalTargets = append(originalTargets, target)
				continue
			}

			intermediateRunners, err := s.getRunners(environmentId, &cloud.ExecutionTarget{
				Match:     target.Match,
				Not:       target.Not,
				Replicate: replicate,
			})
			if err != nil {
				return nil, errors.Wrap(err, "detecting runners for repeating the executions")
			}

			// Filter the runners to ignore labels without values
			runners := make([]map[string]string, 0)
		loop:
			for i := range intermediateRunners {
				for _, k := range replicate {
					if intermediateRunners[i][k] == "" && (target.Match[k] == nil || !slices.Contains(target.Match[k].Labels, "")) {
						continue loop
					}
				}
				runners = append(runners, intermediateRunners[i])
			}

			// Fallback to any runner matching the initial filters
			if len(runners) == 0 {
				selectors = append(selectors, &cloud.ScheduleExecution{
					Selector:      execution.Selector,
					Targets:       []*cloud.ExecutionTarget{{Match: target.Match, Not: target.Not}},
					Config:        execution.Config,
					ExecutionName: execution.ExecutionName, // TODO: what to do when execution name is configured, but multiple requested?
					Tags:          execution.Tags,
				})
				originalTargets = append(originalTargets, target)
				continue
			}

			// Build execution for each combination
			added := make([]map[string]string, 0)
			for _, labels := range runners {
				if slices.ContainsFunc(added, func(m map[string]string) bool {
					return maps.Equal(m, labels)
				}) {
					continue
				}
				added = append(added, labels)
				matcher := make(map[string]*cloud.ExecutionTargetLabels)
				maps.Copy(matcher, target.Match)
				for k, v := range labels {
					if matcher[k] == nil {
						matcher[k] = &cloud.ExecutionTargetLabels{}
					}
					matcher[k] = &cloud.ExecutionTargetLabels{Labels: append(matcher[k].Labels, v)}
				}
				selectors = append(selectors, &cloud.ScheduleExecution{
					Selector:      execution.Selector,
					Targets:       []*cloud.ExecutionTarget{{Match: matcher, Not: target.Not}},
					Config:        execution.Config,
					ExecutionName: execution.ExecutionName, // TODO: what to do when execution name is configured, but multiple requested?
					Tags:          execution.Tags,
				})
				originalTargets = append(originalTargets, target)
				continue
			}
		}
	}
	intermediateSelectors = nil

	// Resolve executions for each selector
	intermediate := make([]*IntermediateExecution, 0, len(selectors))
	for i, v := range selectors {
		workflow, _ := testWorkflows.GetByName(v.Selector.Name)
		originalTarget := testkube.ExecutionTarget{
			Match: common.MapMap(originalTargets[i].Match, func(t *cloud.ExecutionTargetLabels) []string {
				return t.Labels
			}),
			Not: common.MapMap(originalTargets[i].Not, func(t *cloud.ExecutionTargetLabels) []string {
				return t.Labels
			}),
			Replicate: originalTargets[i].Replicate,
		}
		target := originalTarget
		if len(v.Targets) == 1 {
			target = testkube.ExecutionTarget{
				Match: common.MapMap(v.Targets[0].Match, func(t *cloud.ExecutionTargetLabels) []string {
					return t.Labels
				}),
				Not: common.MapMap(v.Targets[0].Not, func(t *cloud.ExecutionTargetLabels) []string {
					return t.Labels
				}),
				Replicate: v.Targets[0].Replicate,
			}
		}
		current := base.Clone().
			AutoGenerateID().
			SetName(v.ExecutionName).
			AppendTags(v.Tags).
			SetWorkflow(testworkflows2.MapAPIToKube(workflow)).
			SetTarget(target).
			SetOriginalTarget(originalTarget)
		intermediate = append(intermediate, current)

		// Inject configuration
		storeConfig := true
		schema := workflow.Spec.Config
		for k := range v.Config {
			if s, ok := schema[k]; ok && s.Sensitive {
				storeConfig = false
			}
		}
		if storeConfig && testworkflows.CountMapBytes(v.Config) < ConfigSizeLimit {
			current.StoreConfig(v.Config)
		}

		// Apply the configuration
		if err := current.ApplyConfig(v.Config); err != nil {
			current.SetError("Cannot inline Test Workflow configuration", err)
			continue
		}

		// Load the required Test Workflow Templates
		tpls, err := testWorkflowTemplates.GetMany(current.TemplateNames())
		if err != nil {
			current.SetError("Cannot fetch required Test Workflow Templates", err)
			continue
		}

		// Apply the Test Workflow Templates
		if err = current.ApplyTemplates(tpls); err != nil {
			current.SetError("Cannot inline Test Workflow Templates", err)
			continue
		}
	}

	// Simplify group ID in case of single execution
	if len(intermediate) == 1 {
		intermediate[0].SetGroupID(intermediate[0].ID())
	}

	// Validate if there are no execution name duplicates initially
	if err = ValidateExecutionNameDuplicates(intermediate); err != nil {
		close(ch)
		return ch, err
	}

	// Validate if the static execution names are not reserved in the database already
	for i := range intermediate {
		if intermediate[i].Name() == "" {
			continue
		}
		if err = ValidateExecutionNameRemoteDuplicate(ctx, s.resultsRepository, intermediate[i]); err != nil {
			close(ch)
			return ch, err
		}
	}

	// Ensure the rest of operations won't be stopped if started
	if ctx.Err() != nil {
		close(ch)
		return ch, ctx.Err()
	}
	cancel()

	// Generate execution names and sequence numbers
	for i := range intermediate {
		// Load execution identifier data
		number, err := s.resultsRepository.GetNextExecutionNumber(context.Background(), intermediate[i].WorkflowName())
		if err != nil {
			close(ch)
			return ch, errors.Wrap(err, "registering next execution sequence number")
		}
		intermediate[i].SetSequenceNumber(number)

		// Generating the execution name
		if intermediate[i].Name() == "" {
			name := fmt.Sprintf("%s-%d", intermediate[i].WorkflowName(), number)
			intermediate[i].SetName(name)

			// Edge case: Check for local duplicates, if there is no clash between static and auto-generated one
			if err = ValidateExecutionNameDuplicates(intermediate); err != nil {
				return ch, err
			}

			// Ensure the execution name is unique
			if err = ValidateExecutionNameRemoteDuplicate(context.Background(), s.resultsRepository, intermediate[i]); err != nil {
				close(ch)
				return ch, err
			}
		}

		// Resolve it finally
		err = intermediate[i].Resolve(s.organizationId, environmentId, req.ParentExecutionIds, false)
		if err != nil {
			intermediate[i].SetError("Cannot process Test Workflow specification", err)
			continue
		}
	}

	go func() {
		defer close(ch)
		for i := range intermediate {
			// Prepare sensitive data
			if err = sensitiveDataHandler.Process(intermediate[i]); err != nil {
				intermediate[i].SetError("Cannot store the sensitive data", err)
			}

			// Save empty logs if the execution is already finished
			if intermediate[i].Finished() {
				_ = s.saveEmptyLogs(context.Background(), intermediate[i].Execution())
			}

			// Insert the execution
			if err = s.insert(context.Background(), intermediate[i].Execution()); err != nil {
				sensitiveDataHandler.Rollback(intermediate[i].ID())
				// TODO: notify API about problem (?)
				continue
			}

			// Inform about the next execution
			ch <- intermediate[i].Execution()
		}
	}()

	return ch, nil
}

func (s *scheduler) CriticalError(execution *testkube.TestWorkflowExecution, name string, err error) error {
	execution.InitializationError(name, err)
	_ = s.saveEmptyLogs(context.Background(), execution)
	return s.finish(context.Background(), execution)
}

func (s *scheduler) finish(ctx context.Context, execution *testkube.TestWorkflowExecution) error {
	if !s.newExecutionsEnabled {
		return s.update(ctx, execution)
	}

	md := metadata.New(map[string]string{
		"api-key":         s.grpcApiToken,
		"organization-id": s.organizationId,
		"agent-id":        s.agentId,
		"environment-id":  s.defaultEnvironmentId,
	})
	opts := []grpc.CallOption{grpc.UseCompressor(gzip.Name), grpc.MaxCallRecvMsgSize(math.MaxInt32)}
	resultBytes, err := json.Marshal(execution)
	if err != nil {
		return err
	}
	err = retry(SaveResultRetryMaxAttempts, SaveResultRetryBaseDelay, func() error {
		_, err := s.grpcClient.FinishExecution(metadata.NewOutgoingContext(ctx, md), &cloud.FinishExecutionRequest{
			Id:     execution.Id,
			Result: resultBytes,
		}, opts...)
		if err != nil {
			s.logger.Warnw("failed to finish the TestWorkflow execution in database", "recoverable", true, "executionId", execution.Id, "error", err)
		}
		return err
	})
	if err != nil {
		s.logger.Errorw("failed to finish the TestWorkflow execution in database", "recoverable", false, "executionId", execution.Id, "error", err)
	}
	return err
}

func (s *scheduler) start(ctx context.Context, execution *testkube.TestWorkflowExecution) error {
	if !s.newExecutionsEnabled {
		return s.init(ctx, execution)
	}

	md := metadata.New(map[string]string{
		"api-key":         s.grpcApiToken,
		"organization-id": s.organizationId,
		"agent-id":        s.agentId,
		"environment-id":  s.defaultEnvironmentId,
	})
	opts := []grpc.CallOption{grpc.UseCompressor(gzip.Name), grpc.MaxCallRecvMsgSize(math.MaxInt32)}
	signatureBytes, err := json.Marshal(execution.Signature)
	if err != nil {
		return err
	}
	err = retry(SaveResultRetryMaxAttempts, SaveResultRetryBaseDelay, func() error {
		_, err := s.grpcClient.InitExecution(metadata.NewOutgoingContext(ctx, md), &cloud.InitExecutionRequest{
			Id:        execution.Id,
			Namespace: execution.Namespace,
			Signature: signatureBytes,
		}, opts...)
		if err != nil {
			s.logger.Warnw("failed to init the TestWorkflow execution in database", "recoverable", true, "executionId", execution.Id, "error", err)
		}
		return err
	})
	if err != nil {
		s.logger.Errorw("failed to init the TestWorkflow execution in database", "recoverable", false, "executionId", execution.Id, "error", err)
	}
	return err
}

func (s *scheduler) Start(execution *testkube.TestWorkflowExecution) error {
	return s.start(context.Background(), execution)
}
