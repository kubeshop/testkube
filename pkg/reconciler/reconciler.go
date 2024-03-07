package reconciler

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	executorv1 "github.com/kubeshop/testkube-operator/api/executor/v1"
	executorsclientv1 "github.com/kubeshop/testkube-operator/pkg/client/executors/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executor"
	"github.com/kubeshop/testkube/pkg/mapper/tests"
	"github.com/kubeshop/testkube/pkg/repository/result"
	"github.com/kubeshop/testkube/pkg/repository/testresult"
)

const (
	reconciliationInterval = 5 * time.Minute
	delayTimeout           = time.Minute
)

var (
	errTestAbnoramallyTerminated = errors.New("test execution was abnormally terminated")
)

type Client struct {
	k8sclient            kubernetes.Interface
	resultRepository     result.Repository
	testResultRepository testresult.Repository
	executorsClient      *executorsclientv1.ExecutorsClient
	logger               *zap.SugaredLogger
}

func NewClient(k8sclient kubernetes.Interface, resultRepository result.Repository, testResultRepository testresult.Repository,
	executorsClient *executorsclientv1.ExecutorsClient, logger *zap.SugaredLogger) *Client {
	return &Client{
		k8sclient:            k8sclient,
		resultRepository:     resultRepository,
		testResultRepository: testResultRepository,
		executorsClient:      executorsClient,
		logger:               logger,
	}
}

func (client *Client) Run(ctx context.Context) error {
	client.logger.Debugw("reconciliation started")

	ticker := time.NewTicker(reconciliationInterval)

	defer func() {
		ticker.Stop()
	}()

	for {
		select {
		case <-ticker.C:
			if err := client.ProcessTests(ctx); err != nil {
				client.logger.Errorw("error processing tests statuses", "error", err)
			}

			if err := client.ProcessTestSuites(ctx); err != nil {
				client.logger.Errorw("error processing test suites statuses", "error", err)
			}
		case <-ctx.Done():
			client.logger.Debugw("reconciliation finished")
			return ctx.Err()
		}
	}
}

func (client *Client) ProcessTests(ctx context.Context) error {
	executions, err := client.resultRepository.GetExecutions(ctx,
		result.NewExecutionsFilter().WithStatus(string(*testkube.ExecutionStatusRunning)))
	if err != nil {
		return err
	}

OuterLoop:
	for _, execution := range executions {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			exec, err := client.executorsClient.GetByType(execution.TestType)
			if err != nil {
				client.logger.Errorw("error getting executor by test type", "error", err)
			}

			errMessage := errTestAbnoramallyTerminated.Error()
			id := execution.Id
			pods, err := executor.GetJobPods(ctx, client.k8sclient.CoreV1().Pods(execution.TestNamespace), id, 1, 10)
			if err == nil {
			ExecutorLoop:
				for _, pod := range pods.Items {
					if pod.Labels["job-name"] == id {
						switch pod.Status.Phase {
						case corev1.PodFailed:
							errMessage = pod.Status.Message
							break ExecutorLoop
						default:
							continue OuterLoop
						}
					}
				}
			} else if exec != nil && exec.Spec.ExecutorType == executorv1.ExecutorTypeContainer {
				supportArtifacts := false
				for _, feature := range exec.Spec.Features {
					if feature == executorv1.FeatureArtifacts {
						supportArtifacts = true
						break
					}
				}

				if supportArtifacts && execution.ArtifactRequest != nil && execution.ArtifactRequest.StorageClassName != "" {
					id = execution.Id + "-scraper"
					pods, err = executor.GetJobPods(ctx, client.k8sclient.CoreV1().Pods(execution.TestNamespace), id, 1, 10)
					if err == nil {
					ScraperLoop:
						for _, pod := range pods.Items {
							if pod.Labels["job-name"] == id {
								switch pod.Status.Phase {
								case corev1.PodFailed:
									errMessage = pod.Status.Message
									break ScraperLoop
								default:
									continue OuterLoop
								}
							}
						}
					}
				}
			}

			execution.ExecutionResult = &testkube.ExecutionResult{
				Status:       testkube.ExecutionStatusFailed,
				ErrorMessage: errMessage,
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
				if step.StartTime.IsZero() {
					continue
				}

				var stepDuration time.Duration
				for _, execute := range step.Execute {
					if execute.Step != nil {
						if execute.Step.Type() == testkube.TestSuiteStepTypeExecuteTest && execute.Execution != nil {
							exec, err := client.resultRepository.Get(ctx, execute.Execution.Id)
							if err != nil && err != mongo.ErrNoDocuments {
								return err
							}

							if exec.ExecutionResult == nil {
								continue
							}

							if exec.ExecutionResult.IsRunning() || exec.ExecutionResult.IsQueued() {
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

						if execute.Step.Type() == testkube.TestSuiteStepTypeDelay {
							duration, err := time.ParseDuration(execute.Step.Delay)
							if err != nil {
								return err
							}

							if duration > stepDuration {
								stepDuration = duration
							}
						}
					}
				}

				if step.EndTime.IsZero() && stepDuration != 0 {
					if time.Since(step.StartTime) < stepDuration+delayTimeout {
						continue OuterLoop
					}

					status = testkube.TestSuiteExecutionStatusFailed
				}
			}

			execution.Status = status
			for i := range execution.ExecuteStepResults {
				for j := range execution.ExecuteStepResults[i].Execute {
					if execution.ExecuteStepResults[i].Execute[j].Execution != nil &&
						execution.ExecuteStepResults[i].Execute[j].Execution.IsRunning() {
						execution.ExecuteStepResults[i].Execute[j].Execution.ExecutionResult = &testkube.ExecutionResult{
							Status:       tests.MapTestSuiteExecutionStatusToExecutionStatus(status),
							ErrorMessage: errTestAbnoramallyTerminated.Error(),
						}
					}
				}
			}
			if err = client.testResultRepository.Update(ctx, execution); err != nil {
				return err
			}
		}
	}

	return nil
}
