package reconciler

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/repository/result"
	"github.com/kubeshop/testkube/pkg/repository/testresult"
)

type Client struct {
	resultRepository     result.Repository
	testResultRepository testresult.Repository
	logger               *zap.SugaredLogger
}

func NewClient(resultRepository result.Repository, testResultRepository testresult.Repository,
	logger *zap.SugaredLogger) (*Client, error) {
	return &Client{
		resultRepository:     resultRepository,
		testResultRepository: testResultRepository,
		logger:               logger,
	}, nil
}

func (client *Client) Run(ctx context.Context) {
	client.logger.Debugw("reconciliation started")

	timer := time.NewTimer(5 * time.Minute)

	defer func() {
		timer.Stop()
	}()

	for {
		select {
		case <-timer.C:
			if err := client.ProcessTests(ctx); err != nil {
				client.logger.Errorw("error processing tests statuses %w", err)
				break
			}

			if err := client.ProcessTestSuites(ctx); err != nil {
				client.logger.Errorw("error processing test suites statuses %w", err)
				break
			}
		case <-ctx.Done():
			client.logger.Debugw("reconciliation finished")
			return
		}
	}
}

func (client *Client) ProcessTests(ctx context.Context) error {
	executions, err := client.resultRepository.GetExecutions(ctx,
		result.NewExecutionsFilter().WithStatus(string(*testkube.ExecutionStatusRunning)))
	if err != nil {
		return err
	}

	for _, execution := range executions {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if true {
				continue
			}

			execution.ExecutionResult = &testkube.ExecutionResult{
				Status:       testkube.ExecutionStatusFailed,
				ErrorMessage: "testkube api server crashed",
			}
			if err = client.resultRepository.Update(ctx, execution); err != nil {
				return err
			}
		}
	}

	return nil
}

func (client *Client) ProcessTestSuites(ctx context.Context) error {
	executions, err := client.testResultRepository.GetExecutions(ctx,
		testresult.NewExecutionsFilter().WithStatus(string(*testkube.TestSuiteExecutionStatusRunning)))
	if err != nil {
		return err
	}

OuterLoop:
	for _, execution := range executions {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			status := testkube.TestSuiteExecutionStatusPassed
		InnerLoop:
			for _, step := range execution.ExecuteStepResults {
				for _, execute := range step.Execute {
					if execute.Step != nil && execute.Step.Type() == testkube.TestSuiteStepTypeExecuteTest {
						exec, err := client.resultRepository.Get(ctx, execute.Execution.Id)
						if err != nil && err != mongo.ErrNoDocuments {
							return err
						}

						if exec.ExecutionResult == nil {
							continue OuterLoop
						}

						if exec.ExecutionResult.IsRunning() {
							continue OuterLoop
						}

						if exec.ExecutionResult.IsFailed() {
							status = testkube.TestSuiteExecutionStatusFailed
						}

						if exec.ExecutionResult.IsAborted() {
							status = testkube.TestSuiteExecutionStatusAborted
							break InnerLoop
						}

						if exec.ExecutionResult.IsTimeout() {
							status = testkube.TestSuiteExecutionStatusTimeout
							break InnerLoop
						}
					}
				}
			}

			execution.Status = status
			if err = client.testResultRepository.Update(ctx, execution); err != nil {
				return err
			}
		}
	}

	return nil
}
