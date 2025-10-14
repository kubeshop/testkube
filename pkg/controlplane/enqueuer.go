package controlplane

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/avast/retry-go/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	testworkflows2 "github.com/kubeshop/testkube/pkg/mapper/testworkflows"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowclient"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowtemplateclient"
	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowexecutor"
)

type Enqueuer struct {
	logger              *zap.SugaredLogger
	workflowRepository  testworkflowclient.TestWorkflowClient
	templateRepository  testworkflowtemplateclient.TestWorkflowTemplateClient
	executionRepository testworkflow.Repository
}

func NewEnqueuer(
	logger *zap.SugaredLogger,
	workflowRepository testworkflowclient.TestWorkflowClient,
	templateRepository testworkflowtemplateclient.TestWorkflowTemplateClient,
	executionRepository testworkflow.Repository,
) Enqueuer {
	return Enqueuer{
		logger:              logger,
		workflowRepository:  workflowRepository,
		templateRepository:  templateRepository,
		executionRepository: executionRepository,
	}
}

type WorkflowFetcher interface {
	Get(selector *cloud.ScheduleResourceSelector) ([]*testkube.TestWorkflow, error)
	GetByName(name string) (*testkube.TestWorkflow, error)
}

type TemplateFetcher interface {
	Get(name string) (*testkube.TestWorkflowTemplate, error)
	GetMany(names map[string]struct{}) (map[string]*testkube.TestWorkflowTemplate, error)
}

func (e *Enqueuer) Execute(ctx context.Context, req *cloud.ScheduleRequest) ([]testkube.TestWorkflowExecution, error) {
	if len(req.Executions) == 0 {
		return []testkube.TestWorkflowExecution{}, nil
	}
	if err := testworkflowexecutor.ValidateExecutionRequest(req); err != nil {
		return nil, err
	}

	logger := e.logger.With("service", "enqueuer")

	executionRequestsByWorkflowName, err := e.resolveExecutionLabelSelectors(ctx, req.Executions)
	if err != nil {
		return nil, err
	}

	// Note: Execution targets do not need to be normalised, since Standalone Testkube only has a single runner.
	// Note: Executions do not need to be replicated, since Standalone Testkube only has a single runner.

	intermediateExecutions, err := e.prepareExecutions(ctx, req, executionRequestsByWorkflowName)
	if err != nil {
		return nil, err
	}

	// START PRE-COMMIT VALIDATION
	// Validate if there are no execution name duplicates initially
	if err = testworkflowexecutor.ValidateExecutionNameDuplicates(intermediateExecutions); err != nil {
		return nil, err
	}

	// Validate if the static execution names are not reserved in the database already
	for i := range intermediateExecutions {
		if intermediateExecutions[i].Name() == "" {
			continue
		}
		if err = testworkflowexecutor.ValidateExecutionNameRemoteDuplicate(ctx, e.executionRepository, intermediateExecutions[i]); err != nil {
			return nil, err
		}
	}

	// Note: Standalone deployment does not support environment queue limits

	// END PRE-COMMIT VALIDATION

	// Ensure the rest of operations won't be stopped if started
	// After this point the operation will be committed even if the context is cancelled.
	// We want to avoid that half executions are commit or that we get gaps within sequence numbers
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	commitContext := context.Background()

	// Generate execution names and sequence numbers
	err = e.finaliseExecution(commitContext, intermediateExecutions)
	if err != nil {
		return nil, err
	}

	// START POST-COMMIT VALIDATION
	// Validate whether the execution can be resolved. Fail-fast in case it cannot.
	for i := range intermediateExecutions {
		exec := intermediateExecutions[i]
		err = exec.Resolve(common.StandaloneOrganization, common.StandaloneOrganizationSlug, common.StandaloneEnvironment, common.StandaloneEnvironmentSlug, req.ParentExecutionIds, false)
		if err != nil {
			exec.SetError("Cannot process Test Workflow specification", err)
			continue
		}
	}
	// END POST-COMMIT VALIDATION

	executions, err := e.persistExecution(commitContext, intermediateExecutions, logger)
	if err != nil {
		return nil, err
	}

	return executions, nil
}

// resolveExecutionLabelSelectors resolve the exact workflows which will get executed.
//
// ScheduleExecution has a `selector` which determines __what__ to execute (i.e. which workflows).
// You can either select workflow execution by `name` or `labels`.
// The resolving does the following transforms _one_ scheduled execution with `labels` selector into _many_ scheduled execution with `name` selectors.
func (e *Enqueuer) resolveExecutionLabelSelectors(ctx context.Context, executions []*cloud.ScheduleExecution) ([]*cloud.ScheduleExecution, error) {
	var result []*cloud.ScheduleExecution

	for _, execution := range executions {
		list, err := e.fetchWorkflow(ctx, execution.Selector)
		if err != nil {
			return nil, err
		}
		for _, w := range list {
			result = append(result, &cloud.ScheduleExecution{
				Selector:      &cloud.ScheduleResourceSelector{Name: w.Name},
				Targets:       execution.Targets,
				Config:        execution.Config,
				Runtime:       execution.Runtime,
				ExecutionName: execution.ExecutionName, // TODO: what to do when execution name is configured, but multiple requested?
				Tags:          execution.Tags,
			})
		}
	}

	return result, nil
}

func (e *Enqueuer) fetchWorkflow(ctx context.Context, selector *cloud.ScheduleResourceSelector) ([]testkube.TestWorkflow, error) {
	if selector.Name == "" {
		return e.workflowRepository.List(ctx, common.StandaloneEnvironment, testworkflowclient.ListOptions{Labels: selector.Labels})
	}
	v, err := e.workflowRepository.Get(ctx, common.StandaloneEnvironment, selector.Name)
	if err != nil {
		return nil, err
	}
	return []testkube.TestWorkflow{*v}, nil
}

// prepareExecutions scaffolds the execution with its targets, workflow, inject & apply config, apply templates etc.
// It does not yet populate the execution sequence number or derived execution name.
// This allows queue limits and policies to drop the prepared execution without sequence number gaps.
func (e *Enqueuer) prepareExecutions(ctx context.Context, req *cloud.ScheduleRequest, executions []*cloud.ScheduleExecution) ([]*testworkflowexecutor.IntermediateExecution, error) {
	result := make([]*testworkflowexecutor.IntermediateExecution, 0, len(executions))

	now := time.Now().UTC()
	executionBase := testworkflowexecutor.NewIntermediateExecution().
		SetGroupID(primitive.NewObjectIDFromTimestamp(now).Hex()).
		SetScheduledAt(now).
		AppendTags(req.Tags).
		SetDisabledWebhooks(req.DisableWebhooks).
		SetKubernetesObjectName(req.KubernetesObjectName).
		SetRunningContext(testworkflowexecutor.GetLegacyRunningContext(req))

	hasResolvedWorkflow := len(req.ResolvedWorkflow) != 0
	if hasResolvedWorkflow {
		var workflow testkube.TestWorkflow
		if err := json.Unmarshal(req.ResolvedWorkflow, &workflow); err != nil {
			return nil, err
		}
		executionBase.SetWorkflow(testworkflows2.MapAPIToKube(&workflow))
	}

	for _, exec := range executions {
		var workflow *testkube.TestWorkflow
		if !hasResolvedWorkflow {
			workflow, _ = e.workflowRepository.Get(ctx, common.StandaloneEnvironment, exec.Selector.Name)
		}

		// Note: Standalone does not support targeting so override to empty targets
		originalTarget := testkube.ExecutionTarget{}
		target := testkube.ExecutionTarget{}

		current := executionBase.Clone().
			AutoGenerateID().
			SetName(exec.ExecutionName).
			AppendTags(exec.Tags).
			SetTarget(target).
			SetOriginalTarget(originalTarget)

		if !hasResolvedWorkflow {
			current.SetWorkflow(testworkflows2.MapAPIToKube(workflow))
		}

		result = append(result, current)

		if exec.Runtime != nil && len(exec.Runtime.EnvVars) > 0 {
			current.SetVariables(exec.Runtime.EnvVars)
		}

		// Inject configuration
		if countMapBytes(exec.Config) < testworkflowexecutor.ConfigSizeLimit {
			current.StoreConfig(exec.Config)
		}

		// Apply the configuration
		if err := current.ApplyConfig(exec.Config); err != nil {
			current.SetError("Cannot inline Test Workflow configuration", err)
			continue
		}

		if !hasResolvedWorkflow {
			// Load the required Test Workflow Templates
			templates, err := e.fetchTemplates(ctx, current.TemplateNames())
			if err != nil {
				current.SetError("Cannot fetch required Test Workflow Templates", err)
				continue
			}

			// Apply the Test Workflow Templates
			if err = current.ApplyTemplates(templates); err != nil {
				current.SetError("Cannot inline Test Workflow Templates", err)
				continue
			}
		}
	}

	// Simplify group ID in case of single execution
	// note: by default the groupID is generated in the executionBase
	if len(result) == 1 {
		result[0].SetGroupID(result[0].ID())
	}

	return result, nil
}

func (e *Enqueuer) fetchTemplates(ctx context.Context, names map[string]struct{}) (map[string]*testkube.TestWorkflowTemplate, error) {
	templates := map[string]*testkube.TestWorkflowTemplate{}
	for name := range names {
		t, err := e.templateRepository.Get(ctx, common.StandaloneEnvironment, name)
		if err != nil {
			return nil, err
		}
		templates[name] = t
	}
	return templates, nil
}

// finaliseExecution will make the executions ready to be committed.
// It will generate the sequence numbers and data derived from this.
// Once the sequence number is generated, the execution must be persisted to the database as gaps in sequence numbers are faulty.
func (e *Enqueuer) finaliseExecution(ctx context.Context, executions []*testworkflowexecutor.IntermediateExecution) error {
	for i := range executions {
		exec := executions[i]
		// Load execution identifier data
		number, err := e.executionRepository.GetNextExecutionNumber(ctx, exec.WorkflowName())
		if err != nil {
			return fmt.Errorf("registering next exec sequence number: %w", err)
		}
		exec.SetSequenceNumber(number)

		// Generating the execution name
		if exec.Name() == "" {
			name := fmt.Sprintf("%s-%d", exec.WorkflowName(), number)
			if len(executions) > 1 {
				name = fmt.Sprintf("%s-%d-%d", exec.WorkflowName(), exec.SequenceNumber(), i+1)
			}
			exec.SetName(name)

			// Edge case: Check for local duplicates, if there is no clash between static and auto-generated one
			if err = testworkflowexecutor.ValidateExecutionNameDuplicates(executions); err != nil {
				return err
			}

			// Ensure the execution name is unique
			if err = testworkflowexecutor.ValidateExecutionNameRemoteDuplicate(ctx, e.executionRepository, exec); err != nil {
				return err
			}
		}
	}

	return nil
}

// persistExecution will persist the execution to the database.
//
// Note: persistExecution should not fail. All passed executions MUST be handled,
// it is not acceptable for persistExecution to partially persist passed executions,
// nor is it possible to rollback once this function is called.
// The current implementation does permit a partial insert because requiring a full
// insert with no rollback is not possible using the current database implementation.
// WARNING: This implementation can result in skipped execution numbers!
func (e *Enqueuer) persistExecution(ctx context.Context, executions []*testworkflowexecutor.IntermediateExecution, logger *zap.SugaredLogger) ([]testkube.TestWorkflowExecution, error) {
	var result []testkube.TestWorkflowExecution
	for i := range executions {
		exec := executions[i]

		// Insert the execution
		err := retry.Do(
			func() error {
				err := e.executionRepository.Insert(ctx, *exec.Execution())
				if err != nil {
					logger.Warnw("failed to update the TestWorkflow exec in database", "recoverable", true, "executionId", exec.Execution().Id, "error", err)
				}
				return err
			},
			retry.DelayType(retry.FixedDelay),
			retry.Delay(300*time.Millisecond),
			retry.Attempts(5),
		)
		if err != nil {
			logger.Errorw("failed to update the TestWorkflow exec in database", "recoverable", false, "executionId", exec.Execution().Id, "error", err)
		}

		result = append(result, *exec.Execution())
	}

	return result, nil
}

func countMapBytes(m map[string]string) int {
	totalBytes := 0
	for k, v := range m {
		totalBytes += len(k) + len(v)
	}
	return totalBytes
}
