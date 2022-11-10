package containerexecutor

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/testkube/internal/pkg/api"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/config"
	"github.com/kubeshop/testkube/pkg/executor"
	"github.com/kubeshop/testkube/pkg/executor/client"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/k8sclient"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/telemetry"
)

const (
	pollTimeout  = 24 * time.Hour
	pollInterval = 200 * time.Millisecond
	repoPath     = "/data/repo"
)

type ResultRepository interface {
	// UpdateExecution updates result in execution
	UpdateResult(ctx context.Context, id string, execution testkube.ExecutionResult) error
	// StartExecution updates execution start time
	StartExecution(ctx context.Context, id string, startTime time.Time) error
	// EndExecution updates execution end time
	EndExecution(ctx context.Context, execution testkube.Execution) error
}

type EventEmitter interface {
	Notify(event testkube.Event)
}

// NewContainerExecutor creates new job executor
func NewContainerExecutor(repo ResultRepository, namespace, initImage, jobTemplate string,
	metrics ExecutionCounter, emiter EventEmitter, configMap config.Repository) (client *ContainerExecutor, err error) {
	clientSet, err := k8sclient.ConnectToK8s()
	if err != nil {
		return client, err
	}

	return &ContainerExecutor{
		clientSet:   clientSet,
		repository:  repo,
		log:         log.DefaultLogger,
		namespace:   namespace,
		initImage:   initImage,
		jobTemplate: jobTemplate,
		configMap:   configMap,
		metrics:     metrics,
		emitter:     emiter,
	}, nil
}

type ExecutionCounter interface {
	IncExecuteTest(execution testkube.Execution)
}

// ContainerExecutor is container for managing job executor dependencies
type ContainerExecutor struct {
	repository  ResultRepository
	log         *zap.SugaredLogger
	clientSet   kubernetes.Interface
	namespace   string
	initImage   string
	jobTemplate string
	metrics     ExecutionCounter
	emitter     EventEmitter
	configMap   config.Repository
}

type JobOptions struct {
	Name                  string
	Namespace             string
	Image                 string
	ImagePullSecrets      []string
	Command               []string
	Args                  []string
	WorkingDir            string
	ImageOverride         string
	Jsn                   string
	TestName              string
	JobTemplate           string
	InitImage             string
	SecretEnvs            map[string]string
	Envs                  map[string]string
	HTTPProxy             string
	HTTPSProxy            string
	UsernameSecret        *testkube.SecretRef
	TokenSecret           *testkube.SecretRef
	Variables             map[string]testkube.Variable
	ActiveDeadlineSeconds int64
}

// Logs returns job logs stream channel using kubernetes api
func (c *ContainerExecutor) Logs(id string) (out chan output.Output, err error) {
	out = make(chan output.Output)
	logs := make(chan []byte)

	go func() {
		defer func() {
			c.log.Debug("closing ContainerExecutor.Logs out log")
			close(out)
		}()

		if err := TailJobLogs(c.log, c.clientSet, c.namespace, id, logs); err != nil {
			out <- output.NewOutputError(err)
			return
		}

		for l := range logs {
			entry := output.NewOutputLine(l)
			out <- entry
		}
	}()

	return
}

// Execute starts new external test execution, reads data and returns ID
// Execution is started asynchronously client can check later for results
func (c *ContainerExecutor) Execute(execution *testkube.Execution, options client.ExecuteOptions) (testkube.ExecutionResult, error) {
	result := testkube.NewRunningExecutionResult()

	ctx := context.Background()
	err := c.createJob(ctx, *execution, options)
	if err != nil {
		return result.Err(err), err
	}

	podsClient := c.clientSet.CoreV1().Pods(c.namespace)
	pods, err := executor.GetJobPods(podsClient, execution.Id, 1, 10)
	if err != nil {
		return result.Err(err), err
	}

	l := c.log.With("executionID", execution.Id, "type", "async")

	for _, pod := range pods.Items {
		if pod.Status.Phase != corev1.PodRunning && pod.Labels["job-name"] == execution.Id {
			// async wait for complete status or error
			go func(pod corev1.Pod) {
				_, err := c.updateResultsFromPod(ctx, pod, l, execution, result)
				if err != nil {
					l.Errorw("update results from jobs pod error", "error", err)
				}
			}(pod)

			return result, nil
		}
	}

	l.Debugw("no pods was found", "totalPodsCount", len(pods.Items))

	return testkube.NewRunningExecutionResult(), nil
}

// Execute starts new external test execution, reads data and returns ID
// Execution is started synchronously client will be blocked
func (c *ContainerExecutor) ExecuteSync(execution *testkube.Execution, options client.ExecuteOptions) (testkube.ExecutionResult, error) {
	result := testkube.NewRunningExecutionResult()

	ctx := context.Background()
	err := c.createJob(ctx, *execution, options)
	if err != nil {
		return result.Err(err), err
	}

	podsClient := c.clientSet.CoreV1().Pods(c.namespace)
	pods, err := executor.GetJobPods(podsClient, execution.Id, 1, 10)
	if err != nil {
		return result.Err(err), err
	}

	l := c.log.With("executionID", execution.Id, "type", "sync")

	// get job pod and
	for _, pod := range pods.Items {
		if pod.Status.Phase != corev1.PodRunning && pod.Labels["job-name"] == execution.Id {
			return c.updateResultsFromPod(ctx, pod, l, execution, result)
		}
	}

	l.Debugw("no pods was found", "totalPodsCount", len(pods.Items))

	return result, nil
}

// createJob creates new Kubernetes job based on execution and execute options
func (c *ContainerExecutor) createJob(ctx context.Context, execution testkube.Execution, options client.ExecuteOptions) error {
	jobs := c.clientSet.BatchV1().Jobs(c.namespace)

	jobOptions, err := NewJobOptions(c.initImage, c.jobTemplate, execution, options)
	if err != nil {
		return err
	}

	c.log.Debug("creating job with options", "options", jobOptions)
	jobSpec, err := NewJobSpec(c.log, jobOptions)
	if err != nil {
		return err
	}

	_, err = jobs.Create(ctx, jobSpec, metav1.CreateOptions{})
	return err
}

// updateResultsFromPod watches logs and stores results if execution is finished
func (c *ContainerExecutor) updateResultsFromPod(ctx context.Context, pod corev1.Pod, l *zap.SugaredLogger, execution *testkube.Execution, result testkube.ExecutionResult) (testkube.ExecutionResult, error) {
	var err error

	// save stop time and final state
	defer c.stopExecution(ctx, execution, &result)

	// wait for complete
	l.Debug("poll immediate waiting for pod to succeed")
	if err = wait.PollImmediate(pollInterval, pollTimeout, executor.IsPodReady(c.clientSet, pod.Name, c.namespace)); err != nil {
		// continue on poll err and try to get logs later
		l.Errorw("waiting for pod complete error", "error", err)
	}
	l.Debug("poll immediate end")

	// we need to retrieve the Pod to get it's latest status
	latestPod, err := c.clientSet.CoreV1().Pods(c.namespace).Get(context.Background(), pod.Name, metav1.GetOptions{})
	if err != nil {
		return result, err
	}

	switch latestPod.Status.Phase {
	case corev1.PodSucceeded:
		result.Success()
	case corev1.PodFailed:
		result.Error()
	}

	var logs []byte
	logs, err = executor.GetPodLogs(c.clientSet, c.namespace, pod)
	if err != nil {
		l.Errorw("get pod logs error", "error", err)
		err = c.repository.UpdateResult(ctx, execution.Id, result.Err(err))
		if err != nil {
			l.Infow("Update result", "error", err)
		}
		return result, err
	}

	result.Output = string(logs)

	l.Infow("container execution completed saving result", "executionId", execution.Id, "status", result.Status)
	err = c.repository.UpdateResult(ctx, execution.Id, result)
	if err != nil {
		l.Errorw("Update execution result error", "error", err)
	}
	return result, nil
}

func (c *ContainerExecutor) stopExecution(ctx context.Context, execution *testkube.Execution, result *testkube.ExecutionResult) {
	c.log.Debug("stopping execution")
	execution.Stop()
	err := c.repository.EndExecution(ctx, *execution)
	if err != nil {
		c.log.Errorw("Update execution result error", "error", err)
	}

	// metrics increase
	execution.ExecutionResult = result
	c.metrics.IncExecuteTest(*execution)

	c.emitter.Notify(testkube.NewEventEndTestSuccess(execution))

	telemetryEnabled, err := c.configMap.GetTelemetryEnabled(ctx)
	if err != nil {
		c.log.Debugw("getting telemetry enabled error", "error", err)
	}

	if !telemetryEnabled {
		return
	}

	clusterID, err := c.configMap.GetUniqueClusterId(ctx)
	if err != nil {
		c.log.Debugw("getting cluster id error", "error", err)
	}

	host, err := os.Hostname()
	if err != nil {
		c.log.Debugw("getting hostname error", "hostname", host, "error", err)
	}

	var dataSource string
	if execution.Content != nil {
		dataSource = execution.Content.Type_
	}

	status := ""
	if execution.ExecutionResult != nil {
		status = string(*execution.ExecutionResult.Status)
	}

	out, err := telemetry.SendRunEvent("testkube_api_run_test", telemetry.RunParams{
		AppVersion: api.Version,
		DataSource: dataSource,
		Host:       host,
		ClusterID:  clusterID,
		TestType:   execution.TestType,
		DurationMs: execution.DurationMs,
		Status:     status,
	})
	if err != nil {
		c.log.Debugw("sending run test telemetry event error", "error", err)
	} else {
		c.log.Debugw("sending run test telemetry event", "output", out)
	}

}

// NewJobOptionsFromExecutionOptions compose JobOptions based on ExecuteOptions
func NewJobOptionsFromExecutionOptions(options client.ExecuteOptions) *JobOptions {
	// for args, command and image, HTTP request takes priority, then test spec, then executor
	var args []string
	switch {
	case len(options.Request.Args) != 0:
		args = options.Request.Args

	case options.TestSpec.ExecutionRequest != nil &&
		len(options.TestSpec.ExecutionRequest.Args) != 0:
		args = options.TestSpec.ExecutionRequest.Args

	case len(options.ExecutorSpec.Command) != 0:
		args = options.ExecutorSpec.Args
	}

	var command []string
	switch {
	case len(options.Request.Command) != 0:
		command = options.Request.Command

	case options.TestSpec.ExecutionRequest != nil &&
		len(options.TestSpec.ExecutionRequest.Command) != 0:
		command = options.TestSpec.ExecutionRequest.Command

	case len(options.ExecutorSpec.Command) != 0:
		command = options.ExecutorSpec.Command
	}

	var image string
	switch {
	case options.Request.Image != "":
		image = options.Request.Image

	case options.TestSpec.ExecutionRequest != nil &&
		options.TestSpec.ExecutionRequest.Image != "":
		image = options.TestSpec.ExecutionRequest.Image

	case options.ExecutorSpec.Image != "":
		image = options.ExecutorSpec.Image
	}

	var workingDir string
	if options.TestSpec.Content != nil &&
		options.TestSpec.Content.Repository != nil &&
		options.TestSpec.Content.Repository.WorkingDir != "" {
		workingDir = options.TestSpec.Content.Repository.WorkingDir
		if !filepath.IsAbs(workingDir) {
			workingDir = filepath.Join(repoPath, workingDir)
		}
	}

	return &JobOptions{
		Image:                 image,
		ImagePullSecrets:      options.ImagePullSecretNames,
		Args:                  args,
		Command:               command,
		WorkingDir:            workingDir,
		TestName:              options.TestName,
		Namespace:             options.Namespace,
		SecretEnvs:            options.Request.SecretEnvs,
		HTTPProxy:             options.Request.HttpProxy,
		HTTPSProxy:            options.Request.HttpsProxy,
		UsernameSecret:        options.UsernameSecret,
		TokenSecret:           options.TokenSecret,
		ActiveDeadlineSeconds: options.Request.ActiveDeadlineSeconds,
	}
}

// AbortK8sJob aborts K8S by job name
func (c *ContainerExecutor) Abort(execution *testkube.Execution) *testkube.ExecutionResult {
	return executor.AbortJob(c.clientSet, c.namespace, execution.Id)
}
