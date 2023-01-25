package containerexecutor

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/kubeshop/testkube/pkg/repository/result"

	"github.com/kubeshop/testkube/pkg/version"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"

	executorv1 "github.com/kubeshop/testkube-operator/apis/executor/v1"
	executorsclientv1 "github.com/kubeshop/testkube-operator/client/executors/v1"
	testsv3 "github.com/kubeshop/testkube-operator/client/tests/v3"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/config"
	"github.com/kubeshop/testkube/pkg/executor"
	"github.com/kubeshop/testkube/pkg/executor/client"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/k8sclient"
	"github.com/kubeshop/testkube/pkg/log"
	testsmapper "github.com/kubeshop/testkube/pkg/mapper/tests"
	"github.com/kubeshop/testkube/pkg/telemetry"
)

const (
	pollTimeout             = 24 * time.Hour
	pollInterval            = 200 * time.Millisecond
	jobDefaultDelaySeconds  = 180
	jobArtifactDelaySeconds = 90
	repoPath                = "/data/repo"
)

type EventEmitter interface {
	Notify(event testkube.Event)
}

// TODO: remove duplicated code that was done when created container executor

// NewContainerExecutor creates new job executor
func NewContainerExecutor(
	repo result.Repository,
	namespace string,
	images executor.Images,
	templates executor.Templates,
	serviceAccountName string,
	metrics ExecutionCounter,
	emiter EventEmitter,
	configMap config.Repository,
	executorsClient *executorsclientv1.ExecutorsClient,
	testsClient testsv3.Interface,
) (client *ContainerExecutor, err error) {
	clientSet, err := k8sclient.ConnectToK8s()
	if err != nil {
		return client, err
	}

	return &ContainerExecutor{
		clientSet:          clientSet,
		repository:         repo,
		log:                log.DefaultLogger,
		namespace:          namespace,
		images:             images,
		templates:          templates,
		configMap:          configMap,
		serviceAccountName: serviceAccountName,
		metrics:            metrics,
		emitter:            emiter,
		executorsClient:    executorsClient,
		testsClient:        testsClient,
	}, nil
}

type ExecutionCounter interface {
	IncExecuteTest(execution testkube.Execution)
}

// ContainerExecutor is container for managing job executor dependencies
type ContainerExecutor struct {
	repository         result.Repository
	log                *zap.SugaredLogger
	clientSet          kubernetes.Interface
	namespace          string
	images             executor.Images
	templates          executor.Templates
	metrics            ExecutionCounter
	emitter            EventEmitter
	configMap          config.Repository
	executorsClient    *executorsclientv1.ExecutorsClient
	serviceAccountName string
	testsClient        testsv3.Interface
}

type JobOptions struct {
	Name                      string
	Namespace                 string
	Image                     string
	ImagePullSecrets          []string
	Command                   []string
	Args                      []string
	WorkingDir                string
	ImageOverride             string
	Jsn                       string
	TestName                  string
	InitImage                 string
	ScraperImage              string
	JobTemplate               string
	ScraperTemplate           string
	PVCTemplate               string
	SecretEnvs                map[string]string
	Envs                      map[string]string
	HTTPProxy                 string
	HTTPSProxy                string
	UsernameSecret            *testkube.SecretRef
	TokenSecret               *testkube.SecretRef
	CertificateSecret         string
	Variables                 map[string]testkube.Variable
	ActiveDeadlineSeconds     int64
	ArtifactRequest           *testkube.ArtifactRequest
	ServiceAccountName        string
	DelaySeconds              int
	JobTemplateExtensions     string
	ScraperTemplateExtensions string
}

// Logs returns job logs stream channel using kubernetes api
func (c *ContainerExecutor) Logs(ctx context.Context, id string) (out chan output.Output, err error) {
	out = make(chan output.Output)

	go func() {
		defer func() {
			c.log.Debug("closing ContainerExecutor.Logs out log")
			close(out)
		}()

		execution, err := c.repository.Get(ctx, id)
		if err != nil {
			out <- output.NewOutputError(err)
			return
		}

		exec, err := c.executorsClient.GetByType(execution.TestType)
		if err != nil {
			out <- output.NewOutputError(err)
			return
		}

		supportArtifacts := false
		for _, feature := range exec.Spec.Features {
			if feature == executorv1.FeatureArtifacts {
				supportArtifacts = true
				break
			}
		}

		ids := []string{id}
		if supportArtifacts && execution.ArtifactRequest != nil {
			ids = append(ids, id+"-scraper")
		}

		for _, podName := range ids {
			logs := make(chan []byte)

			if err := TailJobLogs(ctx, c.log, c.clientSet, c.namespace, podName, logs); err != nil {
				out <- output.NewOutputError(err)
				return
			}

			for l := range logs {
				entry := output.NewOutputLine(l)
				out <- entry
			}
		}
	}()

	return
}

// Execute starts new external test execution, reads data and returns ID
// Execution is started asynchronously client can check later for results
func (c *ContainerExecutor) Execute(ctx context.Context, execution *testkube.Execution, options client.ExecuteOptions) (*testkube.ExecutionResult, error) {
	executionResult := testkube.NewRunningExecutionResult()
	execution.ExecutionResult = executionResult

	jobOptions, err := c.createJob(ctx, *execution, options)
	if err != nil {
		executionResult.Err(err)
		return executionResult, err
	}

	podsClient := c.clientSet.CoreV1().Pods(c.namespace)
	pods, err := executor.GetJobPods(ctx, podsClient, execution.Id, 1, 10)
	if err != nil {
		executionResult.Err(err)
		return executionResult, err
	}

	l := c.log.With("executionID", execution.Id, "type", "async")

	for _, pod := range pods.Items {
		if pod.Status.Phase != corev1.PodRunning && pod.Labels["job-name"] == execution.Id {
			// async wait for complete status or error
			go func(pod corev1.Pod) {
				_, err := c.updateResultsFromPod(ctx, pod, l, execution, jobOptions)
				if err != nil {
					l.Errorw("update results from jobs pod error", "error", err)
				}
			}(pod)

			return executionResult, nil
		}
	}

	l.Debugw("no pods was found", "totalPodsCount", len(pods.Items))

	return execution.ExecutionResult, nil
}

// ExecuteSync starts new external test execution, reads data and returns ID
// Execution is started synchronously client will be blocked
func (c *ContainerExecutor) ExecuteSync(ctx context.Context, execution *testkube.Execution, options client.ExecuteOptions) (*testkube.ExecutionResult, error) {
	executionResult := testkube.NewRunningExecutionResult()
	execution.ExecutionResult = executionResult

	jobOptions, err := c.createJob(ctx, *execution, options)
	if err != nil {
		execution.ExecutionResult.Err(err)
		return execution.ExecutionResult, err
	}

	podsClient := c.clientSet.CoreV1().Pods(c.namespace)
	pods, err := executor.GetJobPods(ctx, podsClient, execution.Id, 1, 10)
	if err != nil {
		execution.ExecutionResult.Err(err)
		return execution.ExecutionResult, err
	}

	l := c.log.With("executionID", execution.Id, "type", "sync")

	// get job pod and
	for _, pod := range pods.Items {
		podNotRunning := pod.Status.Phase != corev1.PodRunning
		IsCorrectJob := pod.Labels["job-name"] == execution.Id
		if podNotRunning && IsCorrectJob {
			return c.updateResultsFromPod(ctx, pod, l, execution, jobOptions)
		}
	}

	l.Debugw("no pods was found", "totalPodsCount", len(pods.Items))

	return execution.ExecutionResult, nil
}

// createJob creates new Kubernetes job based on execution and execute options
func (c *ContainerExecutor) createJob(ctx context.Context, execution testkube.Execution, options client.ExecuteOptions) (*JobOptions, error) {
	jobsClient := c.clientSet.BatchV1().Jobs(c.namespace)

	jobOptions, err := NewJobOptions(c.images, c.templates, c.serviceAccountName, execution, options)
	if err != nil {
		return nil, err
	}

	if jobOptions.ArtifactRequest != nil {
		c.log.Debug("creating persistent volume claim with options", "options", jobOptions)
		pvcsClient := c.clientSet.CoreV1().PersistentVolumeClaims(c.namespace)
		pvcSpec, err := NewPersistentVolumeClaimSpec(c.log, jobOptions)
		if err != nil {
			return nil, err
		}

		_, err = pvcsClient.Create(ctx, pvcSpec, metav1.CreateOptions{})
		if err != nil {
			return nil, err
		}
	}

	c.log.Debug("creating executor job with options", "options", jobOptions)
	jobSpec, err := NewExecutorJobSpec(c.log, jobOptions)
	if err != nil {
		return nil, err
	}

	_, err = jobsClient.Create(ctx, jobSpec, metav1.CreateOptions{})
	return jobOptions, err
}

// updateResultsFromPod watches logs and stores results if execution is finished
func (c *ContainerExecutor) updateResultsFromPod(
	ctx context.Context,
	executorPod corev1.Pod,
	l *zap.SugaredLogger,
	execution *testkube.Execution,
	jobOptions *JobOptions,
) (*testkube.ExecutionResult, error) {
	var err error

	// save stop time and final state
	defer c.stopExecution(ctx, execution, execution.ExecutionResult)

	// wait for complete
	l.Debug("poll immediate waiting for executor pod to succeed")
	if err = wait.PollImmediate(pollInterval, pollTimeout, executor.IsPodReady(ctx, c.clientSet, executorPod.Name, c.namespace)); err != nil {
		// continue on poll err and try to get logs later
		l.Errorw("waiting for executor pod complete error", "error", err)
	}
	l.Debug("poll executor immediate end")

	// we need to retrieve the Pod to get its latest status
	podsClient := c.clientSet.CoreV1().Pods(c.namespace)
	latestExecutorPod, err := podsClient.Get(context.Background(), executorPod.Name, metav1.GetOptions{})
	if err != nil {
		return execution.ExecutionResult, err
	}

	var scraperLogs []byte
	if jobOptions.ArtifactRequest != nil {
		c.log.Debug("creating scraper job with options", "options", jobOptions)
		jobsClient := c.clientSet.BatchV1().Jobs(c.namespace)
		scraperSpec, err := NewScraperJobSpec(c.log, jobOptions)
		if err != nil {
			return execution.ExecutionResult, err
		}

		_, err = jobsClient.Create(ctx, scraperSpec, metav1.CreateOptions{})
		if err != nil {
			return execution.ExecutionResult, err
		}

		scraperPodName := execution.Id + "-scraper"
		scraperPods, err := executor.GetJobPods(ctx, podsClient, scraperPodName, 1, 10)
		if err != nil {
			return execution.ExecutionResult, err
		}

		// get scraper job pod and
		for _, scraperPod := range scraperPods.Items {
			if scraperPod.Status.Phase != corev1.PodRunning && scraperPod.Labels["job-name"] == scraperPodName {
				l.Debug("poll immediate waiting for scraper pod to succeed")
				if err = wait.PollImmediate(pollInterval, pollTimeout, executor.IsPodReady(ctx, c.clientSet, scraperPod.Name, c.namespace)); err != nil {
					// continue on poll err and try to get logs later
					l.Errorw("waiting for scraper pod complete error", "error", err)
				}
				l.Debug("poll scraper immediate end")

				latestScraperPod, err := podsClient.Get(context.Background(), scraperPod.Name, metav1.GetOptions{})
				if err != nil {
					return execution.ExecutionResult, err
				}

				pvcsClient := c.clientSet.CoreV1().PersistentVolumeClaims(c.namespace)
				err = pvcsClient.Delete(ctx, execution.Id+"-pvc", metav1.DeleteOptions{})
				if err != nil {
					return execution.ExecutionResult, err
				}

				switch latestScraperPod.Status.Phase {
				case corev1.PodSucceeded:
					execution.ExecutionResult.Success()
				case corev1.PodFailed:
					execution.ExecutionResult.Error()
				}

				scraperLogs, err = executor.GetPodLogs(ctx, c.clientSet, c.namespace, *latestScraperPod)
				if err != nil {
					l.Errorw("get scraper pod logs error", "error", err)
					return execution.ExecutionResult, err
				}

				break
			}
		}
	}

	if !execution.ExecutionResult.IsFailed() {
		switch latestExecutorPod.Status.Phase {
		case corev1.PodSucceeded:
			execution.ExecutionResult.Success()
		case corev1.PodFailed:
			execution.ExecutionResult.Error()
		}
	}

	executorLogs, err := executor.GetPodLogs(ctx, c.clientSet, c.namespace, *latestExecutorPod)
	if err != nil {
		l.Errorw("get executor pod logs error", "error", err)
		execution.ExecutionResult.Err(err)
		err = c.repository.UpdateResult(ctx, execution.Id, *execution)
		if err != nil {
			l.Infow("Update result", "error", err)
		}
		return execution.ExecutionResult, err
	}

	executorLogs = append(executorLogs, scraperLogs...)
	execution.ExecutionResult.Output = string(executorLogs)

	l.Infow("container execution completed saving result", "executionId", execution.Id, "status", execution.ExecutionResult.Status)
	err = c.repository.UpdateResult(ctx, execution.Id, *execution)
	if err != nil {
		l.Errorw("Update execution result error", "error", err)
	}
	return execution.ExecutionResult, nil
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

	test, err := c.testsClient.Get(execution.TestName)
	if err != nil {
		c.log.Errorw("getting test error", "error", err)
	}

	if test != nil {
		test.Status = testsmapper.MapExecutionToTestStatus(execution)
		if err = c.testsClient.UpdateStatus(test); err != nil {
			c.log.Errorw("updating test error", "error", err)
		}
	}

	if result.IsPassed() {
		c.emitter.Notify(testkube.NewEventEndTestSuccess(execution))
	} else if result.IsTimeout() {
		c.emitter.Notify(testkube.NewEventEndTestTimeout(execution))
	} else if result.IsAborted() {
		c.emitter.Notify(testkube.NewEventEndTestAborted(execution))
	} else {
		c.emitter.Notify(testkube.NewEventEndTestFailed(execution))
	}

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
	if execution.ExecutionResult != nil && execution.ExecutionResult.Status != nil {
		status = string(*execution.ExecutionResult.Status)
	}

	out, err := telemetry.SendRunEvent("testkube_api_run_test", telemetry.RunParams{
		AppVersion: version.Version,
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

	supportArtifacts := false
	for _, feature := range options.ExecutorSpec.Features {
		if feature == executorv1.FeatureArtifacts {
			supportArtifacts = true
			break
		}
	}

	var artifactRequest *testkube.ArtifactRequest
	jobDelaySeconds := jobDefaultDelaySeconds
	if supportArtifacts {
		artifactRequest = options.Request.ArtifactRequest
		jobDelaySeconds = jobArtifactDelaySeconds
	}

	return &JobOptions{
		Image:                     image,
		ImagePullSecrets:          options.ImagePullSecretNames,
		Args:                      args,
		Command:                   command,
		WorkingDir:                workingDir,
		TestName:                  options.TestName,
		Namespace:                 options.Namespace,
		SecretEnvs:                options.Request.SecretEnvs,
		HTTPProxy:                 options.Request.HttpProxy,
		HTTPSProxy:                options.Request.HttpsProxy,
		UsernameSecret:            options.UsernameSecret,
		TokenSecret:               options.TokenSecret,
		CertificateSecret:         options.CertificateSecret,
		ActiveDeadlineSeconds:     options.Request.ActiveDeadlineSeconds,
		ArtifactRequest:           artifactRequest,
		DelaySeconds:              jobDelaySeconds,
		JobTemplateExtensions:     options.Request.JobTemplate,
		ScraperTemplateExtensions: options.Request.ScraperTemplate,
	}
}

// Abort K8sJob aborts K8S by job name
func (c *ContainerExecutor) Abort(ctx context.Context, execution *testkube.Execution) (*testkube.ExecutionResult, error) {
	return executor.AbortJob(ctx, c.clientSet, c.namespace, execution.Id)
}
