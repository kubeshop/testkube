package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/kubeshop/testkube/pkg/featureflags"
	"github.com/kubeshop/testkube/pkg/repository/config"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/pkg/version"

	"github.com/kubeshop/testkube/pkg/repository/result"

	"go.uber.org/zap"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/kustomize/kyaml/yaml/merge2"

	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"

	executorv1 "github.com/kubeshop/testkube-operator/api/executor/v1"
	templatesv1 "github.com/kubeshop/testkube-operator/pkg/client/templates/v1"
	testexecutionsv1 "github.com/kubeshop/testkube-operator/pkg/client/testexecutions/v1"
	testsv3 "github.com/kubeshop/testkube-operator/pkg/client/tests/v3"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event"
	"github.com/kubeshop/testkube/pkg/executor"
	"github.com/kubeshop/testkube/pkg/executor/agent"
	"github.com/kubeshop/testkube/pkg/executor/env"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/log"
	logsclient "github.com/kubeshop/testkube/pkg/logs/client"
	"github.com/kubeshop/testkube/pkg/logs/events"
	testexecutionsmapper "github.com/kubeshop/testkube/pkg/mapper/testexecutions"
	testsmapper "github.com/kubeshop/testkube/pkg/mapper/tests"
	"github.com/kubeshop/testkube/pkg/telemetry"
	"github.com/kubeshop/testkube/pkg/utils"
)

const (
	// GitUsernameSecretName is git username secret name
	GitUsernameSecretName = "git-username"
	// GitUsernameEnvVarName is git username environment var name
	GitUsernameEnvVarName = "RUNNER_GITUSERNAME"
	// GitTokenSecretName is git token secret name
	GitTokenSecretName = "git-token"
	// GitTokenEnvVarName is git token environment var name
	GitTokenEnvVarName = "RUNNER_GITTOKEN"
	// SecretTest is a test secret
	SecretTest = "secrets"
	// SecretSource is a source secret
	SecretSource = "source-secrets"

	pollTimeout  = 24 * time.Hour
	pollInterval = 200 * time.Millisecond
	// pollJobStatus is interval for checking if job timeout occurred
	pollJobStatus = 1 * time.Second
	// timeoutIndicator is string that is added to job logs when timeout occurs
	timeoutIndicator = "DeadlineExceeded"

	logsStreamBuffer = 1000
)

// NewJobExecutor creates new job executor
func NewJobExecutor(
	repo result.Repository,
	images executor.Images,
	templates executor.Templates,
	serviceAccountNames map[string]string,
	metrics ExecutionMetric,
	emiter *event.Emitter,
	configMap config.Repository,
	testsClient testsv3.Interface,
	clientset kubernetes.Interface,
	testExecutionsClient testexecutionsv1.Interface,
	templatesClient templatesv1.Interface,
	registry string,
	podStartTimeout time.Duration,
	clusterID string,
	dashboardURI string,
	apiURI string,
	natsURI string,
	debug bool,
	logsStream logsclient.Stream,
	features featureflags.FeatureFlags,
	defaultStorageClassName string,
) (client *JobExecutor, err error) {
	if serviceAccountNames == nil {
		serviceAccountNames = make(map[string]string)
	}

	return &JobExecutor{
		ClientSet:               clientset,
		Repository:              repo,
		Log:                     log.DefaultLogger,
		images:                  images,
		templates:               templates,
		serviceAccountNames:     serviceAccountNames,
		metrics:                 metrics,
		Emitter:                 emiter,
		configMap:               configMap,
		testsClient:             testsClient,
		testExecutionsClient:    testExecutionsClient,
		templatesClient:         templatesClient,
		registry:                registry,
		podStartTimeout:         podStartTimeout,
		clusterID:               clusterID,
		dashboardURI:            dashboardURI,
		apiURI:                  apiURI,
		natsURI:                 natsURI,
		debug:                   debug,
		logsStream:              logsStream,
		features:                features,
		defaultStorageClassName: defaultStorageClassName,
	}, nil
}

type ExecutionMetric interface {
	IncAndObserveExecuteTest(execution testkube.Execution, dashboardURI string)
}

// JobExecutor is container for managing job executor dependencies
type JobExecutor struct {
	Repository              result.Repository
	Log                     *zap.SugaredLogger
	ClientSet               kubernetes.Interface
	Cmd                     string
	images                  executor.Images
	templates               executor.Templates
	serviceAccountNames     map[string]string
	metrics                 ExecutionMetric
	Emitter                 *event.Emitter
	configMap               config.Repository
	testsClient             testsv3.Interface
	testExecutionsClient    testexecutionsv1.Interface
	templatesClient         templatesv1.Interface
	registry                string
	podStartTimeout         time.Duration
	clusterID               string
	dashboardURI            string
	apiURI                  string
	natsURI                 string
	debug                   bool
	logsStream              logsclient.Stream
	features                featureflags.FeatureFlags
	defaultStorageClassName string
}

type JobOptions struct {
	Name                  string
	Namespace             string
	Image                 string
	ImagePullSecrets      []string
	Jsn                   string
	TestName              string
	InitImage             string
	JobTemplate           string
	Envs                  map[string]string
	SecretEnvs            map[string]string
	HTTPProxy             string
	HTTPSProxy            string
	UsernameSecret        *testkube.SecretRef
	TokenSecret           *testkube.SecretRef
	RunnerCustomCASecret  string
	CertificateSecret     string
	AgentAPITLSSecret     string
	Variables             map[string]testkube.Variable
	ActiveDeadlineSeconds int64
	ServiceAccountName    string
	JobTemplateExtensions string
	EnvConfigMaps         []testkube.EnvReference
	EnvSecrets            []testkube.EnvReference
	Labels                map[string]string
	Registry              string
	ClusterID             string
	ArtifactRequest       *testkube.ArtifactRequest
	WorkingDir            string
	ExecutionNumber       int32
	ContextType           string
	ContextData           string
	Debug                 bool
	NatsUri               string
	LogSidecarImage       string
	APIURI                string
	SlavePodTemplate      string
	Features              featureflags.FeatureFlags
	PvcTemplate           string
	PvcTemplateExtensions string
}

// Logs returns job logs stream channel using kubernetes api
func (c *JobExecutor) Logs(ctx context.Context, id, namespace string) (out chan output.Output, err error) {
	out = make(chan output.Output, logsStreamBuffer)
	logs := make(chan []byte, logsStreamBuffer)

	go func() {
		defer func() {
			c.Log.Debug("closing JobExecutor.Logs out log")
			close(out)
		}()

		if err := c.TailJobLogs(ctx, id, namespace, logs); err != nil {
			out <- output.NewOutputError(err)
			return
		}

		for l := range logs {
			out <- output.GetLogEntry(l)
		}
	}()

	return
}

// Execute starts new external test execution, reads data and returns ID
// Execution is started asynchronously client can check later for results
func (c *JobExecutor) Execute(ctx context.Context, execution *testkube.Execution, options ExecuteOptions) (result *testkube.ExecutionResult, err error) {
	result = testkube.NewRunningExecutionResult()
	execution.ExecutionResult = result

	err = c.CreateJob(ctx, *execution, options)
	if err != nil {
		if cErr := c.cleanPVCVolume(ctx, execution); cErr != nil {
			c.Log.Errorw("error deleting pvc volume", "error", cErr)
		}

		return result.Err(err), err
	}

	c.streamLog(ctx, execution.Id, events.NewLog("created kubernetes job").WithSource(events.SourceJobExecutor))

	if !options.Sync {
		go c.MonitorJobForTimeout(ctx, execution.Id, execution.TestNamespace)
	}

	podsClient := c.ClientSet.CoreV1().Pods(execution.TestNamespace)
	pods, err := executor.GetJobPods(ctx, podsClient, execution.Id, 1, 10)
	if err != nil {
		if cErr := c.cleanPVCVolume(ctx, execution); cErr != nil {
			c.Log.Errorw("error deleting pvc volume", "error", cErr)
		}

		return result.Err(err), err
	}

	l := c.Log.With("executionID", execution.Id, "type", "async")

	c.streamLog(ctx, execution.Id, events.NewLog("waiting for pod to spin up").WithSource(events.SourceJobExecutor))

	for _, pod := range pods.Items {
		if pod.Status.Phase != corev1.PodRunning && pod.Labels["job-name"] == execution.Id {
			// for sync block and complete
			if options.Sync {
				return c.updateResultsFromPod(ctx, pod, l, execution, options.Request.NegativeTest)
			}

			// for async start goroutine and return in progress job
			go func(pod corev1.Pod) {
				_, err := c.updateResultsFromPod(ctx, pod, l, execution, options.Request.NegativeTest)
				if err != nil {
					l.Errorw("update results from jobs pod error", "error", err)
				}
			}(pod)

			return result, nil
		}
	}

	l.Debugw("no pods was found", "totalPodsCount", len(pods.Items))

	return result, nil
}

func (c *JobExecutor) MonitorJobForTimeout(ctx context.Context, jobName, namespace string) {
	ticker := time.NewTicker(pollJobStatus)
	l := c.Log.With("jobName", jobName)
	for {
		select {
		case <-ctx.Done():
			l.Infow("context done, stopping job timeout monitor")
			return
		case <-ticker.C:
			jobs, err := c.ClientSet.BatchV1().Jobs(namespace).List(ctx, metav1.ListOptions{LabelSelector: "job-name=" + jobName})
			if err != nil {
				l.Errorw("could not get jobs", "error", err)
				return
			}
			if jobs == nil || len(jobs.Items) == 0 {
				return
			}

			job := jobs.Items[0]

			if job.Status.Succeeded > 0 {
				l.Debugw("job succeeded", "status", "succeded")
				return
			}

			if job.Status.Failed > 0 {
				l.Debugw("job failed")
				if len(job.Status.Conditions) > 0 {
					for _, condition := range job.Status.Conditions {
						l.Infow("job timeout", "condition.reason", condition.Reason)
						if condition.Reason == timeoutIndicator {
							c.Timeout(ctx, jobName)
						}
					}
				}
				return
			}

			if job.Status.Active > 0 {
				continue
			}
		}
	}
}

// CreateJob creates new Kubernetes job based on execution and execute options
func (c *JobExecutor) CreateJob(ctx context.Context, execution testkube.Execution, options ExecuteOptions) error {
	jobs := c.ClientSet.BatchV1().Jobs(execution.TestNamespace)
	jobOptions, err := NewJobOptions(c.Log, c.templatesClient, c.images, c.templates,
		c.serviceAccountNames, c.registry, c.clusterID, c.apiURI, execution, options, c.natsURI, c.debug)
	if err != nil {
		return err
	}

	if jobOptions.ArtifactRequest != nil &&
		(jobOptions.ArtifactRequest.StorageClassName != "" || jobOptions.ArtifactRequest.UseDefaultStorageClassName) {
		c.Log.Debug("creating persistent volume claim with options", "options", jobOptions)
		pvcsClient := c.ClientSet.CoreV1().PersistentVolumeClaims(execution.TestNamespace)
		pvcSpec, err := NewPersistentVolumeClaimSpec(c.Log, NewPVCOptionsFromJobOptions(jobOptions, c.defaultStorageClassName))
		if err != nil {
			return err
		}

		_, err = pvcsClient.Create(ctx, pvcSpec, metav1.CreateOptions{})
		if err != nil {
			return err
		}
	}

	c.Log.Debug("creating job with options", "options", jobOptions)
	jobSpec, err := NewJobSpec(c.Log, jobOptions)
	if err != nil {
		return err
	}

	_, err = jobs.Create(ctx, jobSpec, metav1.CreateOptions{})
	return err
}

func (c *JobExecutor) cleanPVCVolume(ctx context.Context, execution *testkube.Execution) error {
	if execution.ArtifactRequest != nil &&
		(execution.ArtifactRequest.StorageClassName != "" || execution.ArtifactRequest.UseDefaultStorageClassName) {
		pvcsClient := c.ClientSet.CoreV1().PersistentVolumeClaims(execution.TestNamespace)
		if err := pvcsClient.Delete(ctx, execution.Id+"-pvc", metav1.DeleteOptions{}); err != nil {
			return err
		}
	}

	return nil
}

// updateResultsFromPod watches logs and stores results if execution is finished
func (c *JobExecutor) updateResultsFromPod(ctx context.Context, pod corev1.Pod, l *zap.SugaredLogger, execution *testkube.Execution, isNegativeTest bool) (*testkube.ExecutionResult, error) {
	var err error

	// save stop time and final state
	defer func() {
		if err := c.stopExecution(ctx, l, execution, execution.ExecutionResult, isNegativeTest); err != nil {
			c.streamLog(ctx, execution.Id, events.NewErrorLog(err))
			l.Errorw("error stopping execution after updating results from pod", "error", err)
		}

		if err := c.cleanPVCVolume(ctx, execution); err != nil {
			l.Errorw("error cleaning pvc volume", "error", err)
		}
	}()

	// wait for pod to be loggable
	if err = wait.PollUntilContextTimeout(ctx, pollInterval, c.podStartTimeout, true, executor.IsPodLoggable(c.ClientSet, pod.Name, execution.TestNamespace)); err != nil {
		c.streamLog(ctx, execution.Id, events.NewErrorLog(errors.Wrap(err, "can't start test job pod")))
		l.Errorw("waiting for pod started error", "error", err)
	}

	l.Debug("poll immediate waiting for pod")
	// wait for pod
	if err = wait.PollUntilContextTimeout(ctx, pollInterval, pollTimeout, true, executor.IsPodReady(c.ClientSet, pod.Name, execution.TestNamespace)); err != nil {
		// continue on poll err and try to get logs later
		c.streamLog(ctx, execution.Id, events.NewErrorLog(errors.Wrap(err, "can't read data from pod, pod was not completed")))
		l.Errorw("waiting for pod complete error", "error", err)
	}

	if err != nil {
		execution.ExecutionResult.Err(err)
	}
	l.Debug("poll immediate end")

	c.streamLog(ctx, execution.Id, events.NewLog("analyzing test results and artfacts"))

	logs, err := executor.GetPodLogs(ctx, c.ClientSet, execution.TestNamespace, pod)
	if err != nil {
		l.Errorw("get pod logs error", "error", err)
		c.streamLog(ctx, execution.Id, events.NewErrorLog(err))
	}

	// don't attach logs if logs v2 is enabled - they will be streamed through the logs service
	attachLogs := !c.features.LogsV2
	if len(logs) != 0 {
		// parse job output log (JSON stream)
		execution.ExecutionResult, err = output.ParseRunnerOutput(logs, attachLogs)
		if err != nil {
			l.Errorw("parse output error", "error", err)
			c.streamLog(ctx, execution.Id, events.NewErrorLog(errors.Wrap(err, "can't get test execution job output")))
			return execution.ExecutionResult, err
		}
	}

	if execution.ExecutionResult.IsFailed() {
		errorMessage := execution.ExecutionResult.ErrorMessage
		if errorMessage == "" {
			errorMessage = executor.GetPodErrorMessage(ctx, c.ClientSet, &pod)
		}

		execution.ExecutionResult.ErrorMessage = errorMessage

		c.streamLog(ctx, execution.Id, events.NewErrorLog(errors.Wrap(err, "test execution finished with failed state")))
	} else {
		c.streamLog(ctx, execution.Id, events.NewLog("test execution finshed").WithMetadataEntry("status", string(*execution.ExecutionResult.Status)))
	}

	// saving result in the defer function
	return execution.ExecutionResult, nil
}

func (c *JobExecutor) stopExecution(ctx context.Context, l *zap.SugaredLogger, execution *testkube.Execution, result *testkube.ExecutionResult, isNegativeTest bool) error {
	savedExecution, err := c.Repository.Get(ctx, execution.Id)
	if err != nil {
		l.Errorw("get execution error", "error", err)
		return err
	}

	logEvent := events.NewLog().WithSource(events.SourceJobExecutor)

	l.Debugw("stopping execution", "executionId", execution.Id, "status", result.Status, "executionStatus", execution.ExecutionResult.Status, "savedExecutionStatus", savedExecution.ExecutionResult.Status)

	c.streamLog(ctx, execution.Id, logEvent.WithContent("stopping execution"))
	defer c.streamLog(ctx, execution.Id, logEvent.WithContent("execution stopped"))

	if savedExecution.IsCanceled() || savedExecution.IsTimeout() {
		c.streamLog(ctx, execution.Id, logEvent.WithContent("execution is cancelled"))
		return nil
	}

	execution.Stop()
	if isNegativeTest {
		if result.IsFailed() {
			l.Debugw("test run was expected to fail, and it failed as expected", "test", execution.TestName)
			execution.ExecutionResult.Status = testkube.ExecutionStatusPassed
			execution.ExecutionResult.ErrorMessage = ""
			result.Output = result.Output + "\nTest run was expected to fail, and it failed as expected"
		} else {
			l.Debugw("test run was expected to fail - the result will be reversed", "test", execution.TestName)
			execution.ExecutionResult.Status = testkube.ExecutionStatusFailed
			execution.ExecutionResult.ErrorMessage = "negative test error"
			result.Output = result.Output + "\nTest run was expected to fail, the result will be reversed"
		}

		result.Status = execution.ExecutionResult.Status
		result.ErrorMessage = execution.ExecutionResult.ErrorMessage
	}

	err = c.Repository.EndExecution(ctx, *execution)
	if err != nil {
		l.Errorw("Update execution result error", "error", err)
		return err
	}

	eventToSend := testkube.NewEventEndTestSuccess(execution)
	if result.IsAborted() {
		result.Output = result.Output + "\nTest run was aborted manually."
		eventToSend = testkube.NewEventEndTestAborted(execution)
	} else if result.IsTimeout() {
		result.Output = result.Output + "\nTest run was aborted due to timeout."
		eventToSend = testkube.NewEventEndTestTimeout(execution)
	} else if result.IsFailed() {
		eventToSend = testkube.NewEventEndTestFailed(execution)
	}

	// metrics increase
	execution.ExecutionResult = result
	l.Infow("execution ended, saving result", "executionId", execution.Id, "status", result.Status)
	if err = c.Repository.UpdateResult(ctx, execution.Id, *execution); err != nil {
		l.Errorw("Update execution result error", "error", err)
		return err
	}

	test, err := c.testsClient.Get(execution.TestName)
	if err != nil {
		l.Errorw("getting test error", "error", err)
		return err
	}

	test.Status = testsmapper.MapExecutionToTestStatus(execution)
	if err = c.testsClient.UpdateStatus(test); err != nil {
		l.Errorw("updating test error", "error", err)
		return err
	}

	if execution.TestExecutionName != "" {
		testExecution, err := c.testExecutionsClient.Get(execution.TestExecutionName)
		if err != nil {
			l.Errorw("getting test execution error", "error", err)
			return err
		}

		testExecution.Status = testexecutionsmapper.MapAPIToCRD(execution, testExecution.Generation)
		if err = c.testExecutionsClient.UpdateStatus(testExecution); err != nil {
			l.Errorw("updating test execution error", "error", err)
			return err
		}
	}

	c.metrics.IncAndObserveExecuteTest(*execution, c.dashboardURI)
	c.Emitter.Notify(eventToSend)

	telemetryEnabled, err := c.configMap.GetTelemetryEnabled(ctx)
	if err != nil {
		l.Debugw("getting telemetry enabled error", "error", err)
	}

	if !telemetryEnabled {
		return nil
	}

	clusterID, err := c.configMap.GetUniqueClusterId(ctx)
	if err != nil {
		l.Debugw("getting cluster id error", "error", err)
	}

	host, err := os.Hostname()
	if err != nil {
		l.Debugw("getting hostname error", "hostname", host, "error", err)
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
		l.Debugw("sending run test telemetry event error", "error", err)
	} else {
		l.Debugw("sending run test telemetry event", "output", out)
	}

	return nil
}

// NewJobOptionsFromExecutionOptions compose JobOptions based on ExecuteOptions
func NewJobOptionsFromExecutionOptions(options ExecuteOptions) JobOptions {
	labels := map[string]string{
		testkube.TestLabelTestType: utils.SanitizeName(options.TestSpec.Type_),
		testkube.TestLabelExecutor: options.ExecutorName,
		testkube.TestLabelTestName: options.TestName,
	}
	for key, value := range options.Labels {
		labels[key] = value
	}

	contextType := ""
	contextData := ""
	if options.Request.RunningContext != nil {
		contextType = options.Request.RunningContext.Type_
		contextData = options.Request.RunningContext.Context
	}

	var image string
	if options.ExecutorSpec.Image != "" {
		image = options.ExecutorSpec.Image
	}

	if options.TestSpec.ExecutionRequest != nil &&
		options.TestSpec.ExecutionRequest.Image != "" {
		image = options.TestSpec.ExecutionRequest.Image
	}

	if options.Request.Image != "" {
		image = options.Request.Image
	}

	return JobOptions{
		Image:                 image,
		ImagePullSecrets:      options.ImagePullSecretNames,
		JobTemplate:           options.ExecutorSpec.JobTemplate,
		TestName:              options.TestName,
		Namespace:             options.Namespace,
		Envs:                  options.Request.Envs,
		SecretEnvs:            options.Request.SecretEnvs,
		HTTPProxy:             options.Request.HttpProxy,
		HTTPSProxy:            options.Request.HttpsProxy,
		UsernameSecret:        options.UsernameSecret,
		TokenSecret:           options.TokenSecret,
		RunnerCustomCASecret:  options.RunnerCustomCASecret,
		CertificateSecret:     options.CertificateSecret,
		ActiveDeadlineSeconds: options.Request.ActiveDeadlineSeconds,
		JobTemplateExtensions: options.Request.JobTemplate,
		EnvConfigMaps:         options.Request.EnvConfigMaps,
		EnvSecrets:            options.Request.EnvSecrets,
		Labels:                labels,
		ExecutionNumber:       options.Request.Number,
		ContextType:           contextType,
		ContextData:           contextData,
		Features:              options.Features,
		PvcTemplateExtensions: options.Request.PvcTemplate,
	}
}

// TailJobLogs - locates logs for job pod(s)
func (c *JobExecutor) TailJobLogs(ctx context.Context, id, namespace string, logs chan []byte) (err error) {

	podsClient := c.ClientSet.CoreV1().Pods(namespace)

	pods, err := executor.GetJobPods(ctx, podsClient, id, 1, 10)
	if err != nil {
		close(logs)
		return err
	}

	for _, pod := range pods.Items {
		if pod.Labels["job-name"] == id {

			l := c.Log.With("podNamespace", pod.Namespace, "podName", pod.Name, "podStatus", pod.Status)

			switch pod.Status.Phase {

			case corev1.PodRunning:
				l.Debug("tailing pod logs: immediately")
				return c.TailPodLogs(ctx, pod, logs)

			case corev1.PodFailed:
				err := errors.Errorf("can't get pod logs, pod failed: %s/%s", pod.Namespace, pod.Name)
				l.Errorw(err.Error())
				return c.GetLastLogLineError(ctx, pod)

			default:
				l.Debugw("tailing job logs: waiting for pod to be ready")
				if err = wait.PollUntilContextTimeout(ctx, pollInterval, c.podStartTimeout, true, executor.IsPodLoggable(c.ClientSet, pod.Name, namespace)); err != nil {
					l.Errorw("poll immediate error when tailing logs", "error", err)
					return err
				}

				l.Debug("tailing pod logs")
				return c.TailPodLogs(ctx, pod, logs)
			}
		}
	}

	return
}

func (c *JobExecutor) TailPodLogs(ctx context.Context, pod corev1.Pod, logs chan []byte) (err error) {
	var containers []string
	for _, container := range pod.Spec.InitContainers {
		containers = append(containers, container.Name)
	}

	for _, container := range pod.Spec.Containers {
		containers = append(containers, container.Name)
	}

	l := c.Log.With("method", "TailPodLogs", "pod", pod.Name, "namespace", pod.Namespace, "containersCount", len(containers))

	wg := sync.WaitGroup{}
	wg.Add(len(containers))

	for _, container := range containers {
		go func(container string) {
			defer wg.Done()

			podLogOptions := corev1.PodLogOptions{
				Follow:    true,
				Container: container,
			}

			podLogRequest := c.ClientSet.CoreV1().
				Pods(pod.Namespace).
				GetLogs(pod.Name, &podLogOptions)

			stream, err := podLogRequest.Stream(ctx)
			if err != nil {
				l.Errorw("stream error", "error", err)
				return
			}

			reader := bufio.NewReader(stream)

			for {
				b, err := utils.ReadLongLine(reader)
				if err == io.EOF {
					return
				} else if err != nil {
					l.Errorw("scanner error", "error", err)
					return
				}
				l.Debugw("log chunk pushed", "out", string(b), "pod", pod.Name)
				logs <- b
			}
		}(container)
	}

	go func() {
		defer close(logs)
		l.Debugw("waiting for all containers to finish", "containers", containers)
		wg.Wait()
		l.Infow("log stream finished")
	}()

	return
}

// GetPodLogError returns last line as error
func (c *JobExecutor) GetPodLogError(ctx context.Context, pod corev1.Pod) (logsBytes []byte, err error) {
	// error line should be last one
	return executor.GetPodLogs(ctx, c.ClientSet, pod.Namespace, pod, 1)
}

// GetLastLogLineError return error if last line is failed
func (c *JobExecutor) GetLastLogLineError(ctx context.Context, pod corev1.Pod) error {
	l := c.Log.With("pod", pod.Name, "namespace", pod.Namespace)
	errorLog, err := c.GetPodLogError(ctx, pod)
	if err != nil {
		l.Errorw("getPodLogs error", "error", err, "pod", pod)
		return errors.Errorf("getPodLogs error: %v", err)
	}

	l.Debugw("log", "got last log bytes", string(errorLog)) // in case distorted log bytes
	entry := output.GetLogEntry(errorLog)
	l.Infow("got last log entry", "log", entry.String())
	return errors.Errorf("error from last log entry: %s", entry.String())
}

// Abort aborts K8S by job name
func (c *JobExecutor) Abort(ctx context.Context, execution *testkube.Execution) (result *testkube.ExecutionResult, err error) {
	l := c.Log.With("execution", execution.Id)
	result, err = executor.AbortJob(ctx, c.ClientSet, execution.TestNamespace, execution.Id)
	if err != nil {
		l.Errorw("error aborting job", "execution", execution.Id, "error", err)
	}
	l.Debugw("job aborted", "execution", execution.Id, "result", result)
	if err := c.stopExecution(ctx, l, execution, result, false); err != nil {
		l.Errorw("error stopping execution on job executor abort", "error", err)
	}
	return result, nil
}

func (c *JobExecutor) Timeout(ctx context.Context, jobName string) (result *testkube.ExecutionResult) {
	l := c.Log.With("jobName", jobName)
	l.Infow("job timeout")
	execution, err := c.Repository.Get(ctx, jobName)
	if err != nil {
		l.Errorw("error getting execution", "error", err)
		return
	}

	c.streamLog(ctx, execution.Id, events.NewLog("execution took too long, pod deadline exceeded"))

	result = &testkube.ExecutionResult{
		Status: testkube.ExecutionStatusTimeout,
	}
	if err := c.stopExecution(ctx, l, &execution, result, false); err != nil {
		l.Errorw("error stopping execution on job executor timeout", "error", err)
	}

	return
}

func (c *JobExecutor) streamLog(ctx context.Context, id string, log *events.Log) {
	if c.features.LogsV2 {
		c.logsStream.Push(ctx, id, log)
	}
}

// NewJobSpec is a method to create new job spec
func NewJobSpec(log *zap.SugaredLogger, options JobOptions) (*batchv1.Job, error) {
	envManager := env.NewManager()
	secretEnvVars := append(envManager.PrepareSecrets(options.SecretEnvs, options.Variables),
		envManager.PrepareGitCredentials(options.UsernameSecret, options.TokenSecret)...)

	tmpl, err := utils.NewTemplate("job").Funcs(template.FuncMap{"vartypeptrtostring": testkube.VariableTypeString}).
		Parse(options.JobTemplate)
	if err != nil {
		return nil, errors.Errorf("creating job spec from options.JobTemplate error: %v", err)
	}

	options.Jsn = strings.ReplaceAll(options.Jsn, "'", "''")
	var buffer bytes.Buffer
	if err = tmpl.ExecuteTemplate(&buffer, "job", options); err != nil {
		return nil, errors.Errorf("executing job spec template: %v", err)
	}

	var job batchv1.Job
	jobSpec := buffer.String()
	if options.JobTemplateExtensions != "" {
		tmplExt, err := utils.NewTemplate("jobExt").Funcs(template.FuncMap{"vartypeptrtostring": testkube.VariableTypeString}).
			Parse(options.JobTemplateExtensions)
		if err != nil {
			return nil, errors.Errorf("creating job extensions spec from template error: %v", err)
		}

		var bufferExt bytes.Buffer
		if err = tmplExt.ExecuteTemplate(&bufferExt, "jobExt", options); err != nil {
			return nil, errors.Errorf("executing job extensions spec template: %v", err)
		}

		if jobSpec, err = merge2.MergeStrings(bufferExt.String(), jobSpec, false, kyaml.MergeOptions{}); err != nil {
			return nil, errors.Errorf("merging job spec templates: %v", err)
		}
	}

	log.Debug("Job specification", jobSpec)
	decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewBufferString(jobSpec), len(jobSpec))
	if err := decoder.Decode(&job); err != nil {
		return nil, errors.Errorf("decoding job spec error: %v", err)
	}

	for key, value := range options.Labels {
		if job.Labels == nil {
			job.Labels = make(map[string]string)
		}

		job.Labels[key] = value

		if job.Spec.Template.Labels == nil {
			job.Spec.Template.Labels = make(map[string]string)
		}

		job.Spec.Template.Labels[key] = value
	}

	envs := append(executor.RunnerEnvVars, corev1.EnvVar{Name: "RUNNER_CLUSTERID", Value: options.ClusterID})
	if options.ArtifactRequest != nil && options.ArtifactRequest.StorageBucket != "" {
		envs = append(envs, corev1.EnvVar{Name: "RUNNER_BUCKET", Value: options.ArtifactRequest.StorageBucket})
	} else {
		envs = append(envs, corev1.EnvVar{Name: "RUNNER_BUCKET", Value: os.Getenv("STORAGE_BUCKET")})
	}

	envs = append(envs, secretEnvVars...)
	if options.HTTPProxy != "" {
		envs = append(envs, corev1.EnvVar{Name: "HTTP_PROXY", Value: options.HTTPProxy})
	}

	if options.HTTPSProxy != "" {
		envs = append(envs, corev1.EnvVar{Name: "HTTPS_PROXY", Value: options.HTTPSProxy})
	}

	envs = append(envs, envManager.PrepareEnvs(options.Envs, options.Variables)...)
	envs = append(envs, corev1.EnvVar{Name: "RUNNER_WORKINGDIR", Value: options.WorkingDir})
	envs = append(envs, corev1.EnvVar{Name: "RUNNER_EXECUTIONID", Value: options.Name})
	envs = append(envs, corev1.EnvVar{Name: "RUNNER_TESTNAME", Value: options.TestName})
	envs = append(envs, corev1.EnvVar{Name: "RUNNER_EXECUTIONNUMBER", Value: fmt.Sprint(options.ExecutionNumber)})
	envs = append(envs, corev1.EnvVar{Name: "RUNNER_CONTEXTTYPE", Value: options.ContextType})
	envs = append(envs, corev1.EnvVar{Name: "RUNNER_CONTEXTDATA", Value: options.ContextData})
	envs = append(envs, corev1.EnvVar{Name: "RUNNER_APIURI", Value: options.APIURI})

	for i := range job.Spec.Template.Spec.InitContainers {
		job.Spec.Template.Spec.InitContainers[i].Env = append(job.Spec.Template.Spec.InitContainers[i].Env, envs...)
	}

	for i := range job.Spec.Template.Spec.Containers {
		job.Spec.Template.Spec.Containers[i].Env = append(job.Spec.Template.Spec.Containers[i].Env, envs...)
	}

	return &job, nil
}

func NewJobOptions(log *zap.SugaredLogger, templatesClient templatesv1.Interface, images executor.Images,
	templates executor.Templates, serviceAccountNames map[string]string, registry, clusterID, apiURI string,
	execution testkube.Execution, options ExecuteOptions, natsURI string, debug bool) (jobOptions JobOptions, err error) {
	jsn, err := json.Marshal(execution)
	if err != nil {
		return jobOptions, err
	}

	jobOptions = NewJobOptionsFromExecutionOptions(options)
	jobOptions.Name = execution.Id
	jobOptions.Namespace = execution.TestNamespace
	jobOptions.Jsn = string(jsn)
	jobOptions.InitImage = images.Init
	jobOptions.TestName = execution.TestName
	jobOptions.Features = options.Features

	// options needed for Log sidecar
	if options.Features.LogsV2 {
		// TODO pass them from some config? we dont' have any in this context?
		jobOptions.Debug = debug
		jobOptions.NatsUri = natsURI
		jobOptions.LogSidecarImage = images.LogSidecar
	}

	if jobOptions.JobTemplate == "" {
		jobOptions.JobTemplate = templates.Job
	}

	if options.ExecutorSpec.JobTemplateReference != "" {
		template, err := templatesClient.Get(options.ExecutorSpec.JobTemplateReference)
		if err != nil {
			return jobOptions, err
		}

		if template.Spec.Type_ != nil && testkube.TemplateType(*template.Spec.Type_) == testkube.JOB_TemplateType {
			jobOptions.JobTemplate = template.Spec.Body
		} else {
			log.Warnw("Not matched template type", "template", options.ExecutorSpec.JobTemplateReference)
		}
	}

	if options.Request.JobTemplateReference != "" {
		template, err := templatesClient.Get(options.Request.JobTemplateReference)
		if err != nil {
			return jobOptions, err
		}

		if template.Spec.Type_ != nil && testkube.TemplateType(*template.Spec.Type_) == testkube.JOB_TemplateType {
			jobOptions.JobTemplate = template.Spec.Body
		} else {
			log.Warnw("Not matched template type", "template", options.Request.JobTemplateReference)
		}
	}

	jobOptions.Variables = execution.Variables
	serviceAccountName, ok := serviceAccountNames[execution.TestNamespace]
	if !ok {
		return jobOptions, fmt.Errorf("not supported namespace %s", execution.TestNamespace)
	}

	jobOptions.ServiceAccountName = serviceAccountName
	jobOptions.Registry = registry
	jobOptions.ClusterID = clusterID

	supportArtifacts := false
	for _, feature := range options.ExecutorSpec.Features {
		if feature == executorv1.FeatureArtifacts {
			supportArtifacts = true
			break
		}
	}

	if supportArtifacts {
		jobOptions.ArtifactRequest = execution.ArtifactRequest
	}

	workingDir := agent.GetDefaultWorkingDir(executor.VolumeDir, execution)
	if execution.Content != nil && execution.Content.Repository != nil && execution.Content.Repository.WorkingDir != "" {
		workingDir = filepath.Join(executor.VolumeDir, "repo", execution.Content.Repository.WorkingDir)
	}

	jobOptions.WorkingDir = workingDir
	jobOptions.APIURI = apiURI

	jobOptions.SlavePodTemplate = templates.Slave
	if options.Request.SlavePodRequest != nil && options.Request.SlavePodRequest.PodTemplateReference != "" {
		template, err := templatesClient.Get(options.Request.SlavePodRequest.PodTemplateReference)
		if err != nil {
			return jobOptions, err
		}

		if template.Spec.Type_ != nil && testkube.TemplateType(*template.Spec.Type_) == testkube.POD_TemplateType {
			jobOptions.SlavePodTemplate = template.Spec.Body
		} else {
			log.Warnw("Not matched template type", "template", options.Request.SlavePodRequest.PodTemplateReference)
		}
	}

	if options.ExecutorSpec.Slaves != nil {
		slvesConfigs, err := json.Marshal(executor.GetSlavesConfigs(
			images.Init,
			*options.ExecutorSpec.Slaves,
			jobOptions.Registry,
			jobOptions.ServiceAccountName,
			jobOptions.CertificateSecret,
			jobOptions.SlavePodTemplate,
			jobOptions.ImagePullSecrets,
			jobOptions.EnvConfigMaps,
			jobOptions.EnvSecrets,
			int(jobOptions.ActiveDeadlineSeconds),
			testkube.Features(options.Features),
			natsURI,
			images.LogSidecar,
			jobOptions.RunnerCustomCASecret,
		))

		if err != nil {
			return jobOptions, err
		}

		if jobOptions.Variables == nil {
			jobOptions.Variables = make(map[string]testkube.Variable)
		}

		jobOptions.Variables[executor.SlavesConfigsEnv] = testkube.NewBasicVariable(executor.SlavesConfigsEnv, string(slvesConfigs))
	}

	jobOptions.PvcTemplate = templates.PVC
	if options.Request.PvcTemplateReference != "" {
		template, err := templatesClient.Get(options.Request.PvcTemplateReference)
		if err != nil {
			return jobOptions, err
		}

		if template.Spec.Type_ != nil && testkube.TemplateType(*template.Spec.Type_) == testkube.PVC_TemplateType {
			jobOptions.PvcTemplate = template.Spec.Body
		} else {
			log.Warnw("Not matched template type", "template", options.Request.PvcTemplateReference)
		}
	}

	// used for adding custom certificates for Agent (gRPC) API
	jobOptions.AgentAPITLSSecret = options.AgentAPITLSSecret

	return
}

func NewPVCOptionsFromJobOptions(options JobOptions, defaultStorageClassName string) PVCOptions {
	return PVCOptions{
		Name:                    options.Name,
		Namespace:               options.Namespace,
		PvcTemplate:             options.PvcTemplate,
		PvcTemplateExtensions:   options.PvcTemplateExtensions,
		ArtifactRequest:         options.ArtifactRequest,
		DefaultStorageClassName: defaultStorageClassName,
	}
}
