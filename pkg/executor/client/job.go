package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"strings"
	"text/template"
	"time"

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

	testsv3 "github.com/kubeshop/testkube-operator/client/tests/v3"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/config"
	"github.com/kubeshop/testkube/pkg/event"
	"github.com/kubeshop/testkube/pkg/executor"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/k8sclient"
	"github.com/kubeshop/testkube/pkg/log"
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

	pollTimeout  = 24 * time.Hour
	pollInterval = 200 * time.Millisecond
	volumeDir    = "/data"
	// pollJobStatus is interval for checking if job timeout occurred
	pollJobStatus = 1 * time.Second
	// timeoutIndicator is string that is added to job logs when timeout occurs
	timeoutIndicator = "DeadlineExceeded"
)

// NewJobExecutor creates new job executor
func NewJobExecutor(repo result.Repository, namespace string, images executor.Images,
	templates executor.Templates, serviceAccountName string, metrics ExecutionCounter,
	emiter *event.Emitter, configMap config.Repository, testsClient testsv3.Interface) (client *JobExecutor, err error) {
	clientSet, err := k8sclient.ConnectToK8s()
	if err != nil {
		return client, err
	}

	return &JobExecutor{
		ClientSet:          clientSet,
		Repository:         repo,
		Log:                log.DefaultLogger,
		Namespace:          namespace,
		images:             images,
		templates:          templates,
		serviceAccountName: serviceAccountName,
		metrics:            metrics,
		Emitter:            emiter,
		configMap:          configMap,
		testsClient:        testsClient,
	}, nil
}

type ExecutionCounter interface {
	IncExecuteTest(execution testkube.Execution)
}

// JobExecutor is container for managing job executor dependencies
type JobExecutor struct {
	Repository         result.Repository
	Log                *zap.SugaredLogger
	ClientSet          *kubernetes.Clientset
	Namespace          string
	Cmd                string
	images             executor.Images
	templates          executor.Templates
	serviceAccountName string
	metrics            ExecutionCounter
	Emitter            *event.Emitter
	configMap          config.Repository
	testsClient        testsv3.Interface
}

type JobOptions struct {
	Name                  string
	Namespace             string
	Image                 string
	ImagePullSecrets      []string
	ImageOverride         string
	Jsn                   string
	TestName              string
	InitImage             string
	JobTemplate           string
	SecretEnvs            map[string]string
	HTTPProxy             string
	HTTPSProxy            string
	UsernameSecret        *testkube.SecretRef
	TokenSecret           *testkube.SecretRef
	CertificateSecret     string
	Variables             map[string]testkube.Variable
	ActiveDeadlineSeconds int64
	ServiceAccountName    string
	JobTemplateExtensions string
}

// Logs returns job logs stream channel using kubernetes api
func (c *JobExecutor) Logs(ctx context.Context, id string) (out chan output.Output, err error) {
	out = make(chan output.Output)
	logs := make(chan []byte)

	go func() {
		defer func() {
			c.Log.Debug("closing JobExecutor.Logs out log")
			close(out)
		}()

		if err := c.TailJobLogs(ctx, id, logs); err != nil {
			out <- output.NewOutputError(err)
			return
		}

		for l := range logs {
			entry, err := output.GetLogEntry(l)
			if err != nil {
				out <- output.NewOutputError(err)
				return
			}
			out <- entry
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
		return result.Err(err), err
	}
	go c.MonitorJobForTimeout(ctx, execution.Id)

	podsClient := c.ClientSet.CoreV1().Pods(c.Namespace)
	pods, err := executor.GetJobPods(ctx, podsClient, execution.Id, 1, 10)
	if err != nil {
		return result.Err(err), err
	}

	l := c.Log.With("executionID", execution.Id, "type", "async")

	for _, pod := range pods.Items {
		if pod.Status.Phase != corev1.PodRunning && pod.Labels["job-name"] == execution.Id {
			// async wait for complete status or error
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

	return testkube.NewRunningExecutionResult(), nil
}

// ExecuteSync starts new external test execution, reads data and returns ID
// Execution is started synchronously client will be blocked
func (c *JobExecutor) ExecuteSync(ctx context.Context, execution *testkube.Execution, options ExecuteOptions) (result *testkube.ExecutionResult, err error) {
	result = testkube.NewRunningExecutionResult()
	execution.ExecutionResult = result

	err = c.CreateJob(ctx, *execution, options)
	if err != nil {
		return result.Err(err), err
	}

	podsClient := c.ClientSet.CoreV1().Pods(c.Namespace)
	pods, err := executor.GetJobPods(ctx, podsClient, execution.Id, 1, 10)
	if err != nil {
		return result.Err(err), err
	}

	l := c.Log.With("executionID", execution.Id, "type", "sync")

	// get job pod and
	for _, pod := range pods.Items {
		if pod.Status.Phase != corev1.PodRunning && pod.Labels["job-name"] == execution.Id {
			return c.updateResultsFromPod(ctx, pod, l, execution, options.Request.NegativeTest)
		}
	}

	l.Debugw("no pods was found", "totalPodsCount", len(pods.Items))

	return

}

func (c *JobExecutor) MonitorJobForTimeout(ctx context.Context, jobName string) {
	ticker := time.NewTicker(pollJobStatus)
	l := c.Log.With("jobName", jobName)
	for {
		select {
		case <-ctx.Done():
			l.Infow("context done, stopping job timeout monitor")
			return
		case <-ticker.C:
			jobs, err := c.ClientSet.BatchV1().Jobs(c.Namespace).List(ctx, metav1.ListOptions{LabelSelector: "job-name=" + jobName})
			if err != nil {
				l.Errorw("could not get jobs", "error", err)
				return
			}
			if jobs == nil || len(jobs.Items) == 0 {
				return
			}

			job := jobs.Items[0]

			if job.Status.Succeeded > 0 {
				l.Debugw("job succeeded", "status")
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
	jobs := c.ClientSet.BatchV1().Jobs(c.Namespace)
	jobOptions, err := NewJobOptions(c.images.Init, c.templates.Job, c.serviceAccountName, execution, options)
	if err != nil {
		return err
	}

	c.Log.Debug("creating job with options", "options", jobOptions)
	jobSpec, err := NewJobSpec(c.Log, jobOptions)
	if err != nil {
		return err
	}

	_, err = jobs.Create(ctx, jobSpec, metav1.CreateOptions{})
	return err
}

// updateResultsFromPod watches logs and stores results if execution is finished
func (c *JobExecutor) updateResultsFromPod(ctx context.Context, pod corev1.Pod, l *zap.SugaredLogger, execution *testkube.Execution, isNegativeTest bool) (*testkube.ExecutionResult, error) {
	var err error

	// save stop time and final state
	defer func() {
		if err := c.stopExecution(ctx, l, execution, execution.ExecutionResult, isNegativeTest, err); err != nil {
			l.Errorw("error stopping execution after updating results from pod", "error", err)
		}
	}()

	// wait for complete
	l.Debug("poll immediate waiting for pod to succeed")
	if err = wait.PollImmediate(pollInterval, pollTimeout, executor.IsPodReady(ctx, c.ClientSet, pod.Name, c.Namespace)); err != nil {
		// continue on poll err and try to get logs later
		l.Errorw("waiting for pod complete error", "error", err)
	}
	l.Debug("poll immediate end")

	var logs []byte
	logs, err = executor.GetPodLogs(ctx, c.ClientSet, c.Namespace, pod)
	if err != nil {
		l.Errorw("get pod logs error", "error", err)
		return execution.ExecutionResult, err
	}

	// parse job output log (JSON stream)
	execution.ExecutionResult, err = output.ParseRunnerOutput(logs)
	if err != nil {
		l.Errorw("parse output error", "error", err)
		return execution.ExecutionResult, err
	}
	// saving result in the defer function
	return execution.ExecutionResult, nil

}

func (c *JobExecutor) stopExecution(ctx context.Context, l *zap.SugaredLogger, execution *testkube.Execution, result *testkube.ExecutionResult, isNegativeTest bool, passedErr error) error {
	savedExecution, err := c.Repository.Get(ctx, execution.Id)
	if err != nil {
		l.Errorw("get execution error", "error", err)
		return err
	}
	l.Debugw("stopping execution", "executionId", execution.Id, "status", result.Status, "executionStatus", execution.ExecutionResult.Status, "passedError", passedErr, "savedExecutionStatus", savedExecution.ExecutionResult.Status)

	if savedExecution.IsCanceled() || savedExecution.IsTimeout() {
		return nil
	}

	execution.Stop()
	if isNegativeTest {
		if result.IsFailed() {
			l.Infow("test run was expected to fail, and it failed as expected", "test", execution.TestName)
			execution.ExecutionResult.Status = testkube.ExecutionStatusPassed
			result.Status = testkube.ExecutionStatusPassed
			result.Output = result.Output + "\nTest run was expected to fail, and it failed as expected"
		} else {
			l.Infow("test run was expected to fail - the result will be reversed", "test", execution.TestName)
			execution.ExecutionResult.Status = testkube.ExecutionStatusFailed
			result.Status = testkube.ExecutionStatusFailed
			result.Output = result.Output + "\nTest run was expected to fail, the result will be reversed"
		}
	}

	err = c.Repository.EndExecution(ctx, *execution)
	if err != nil {
		l.Errorw("Update execution result error", "error", err)
		return err
	}

	if passedErr != nil {
		result.Err(passedErr)
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

	if test != nil {
		test.Status = testsmapper.MapExecutionToTestStatus(execution)
		if err = c.testsClient.UpdateStatus(test); err != nil {
			l.Errorw("updating test error", "error", err)
			return err
		}
	}

	c.metrics.IncExecuteTest(*execution)
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
	return JobOptions{
		Image:                 options.ExecutorSpec.Image,
		ImageOverride:         options.ImageOverride,
		JobTemplate:           options.ExecutorSpec.JobTemplate,
		TestName:              options.TestName,
		Namespace:             options.Namespace,
		SecretEnvs:            options.Request.SecretEnvs,
		HTTPProxy:             options.Request.HttpProxy,
		HTTPSProxy:            options.Request.HttpsProxy,
		UsernameSecret:        options.UsernameSecret,
		TokenSecret:           options.TokenSecret,
		CertificateSecret:     options.CertificateSecret,
		ActiveDeadlineSeconds: options.Request.ActiveDeadlineSeconds,
		JobTemplateExtensions: options.Request.JobTemplate,
	}
}

// TailJobLogs - locates logs for job pod(s)
func (c *JobExecutor) TailJobLogs(ctx context.Context, id string, logs chan []byte) (err error) {

	podsClient := c.ClientSet.CoreV1().Pods(c.Namespace)

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
				if err = wait.PollImmediate(pollInterval, pollTimeout, executor.IsPodLoggable(ctx, c.ClientSet, pod.Name, c.Namespace)); err != nil {
					l.Errorw("poll immediate error when tailing logs", "error", err)
					return c.GetLastLogLineError(ctx, pod)
				}

				l.Debug("tailing pod logs")
				return c.TailPodLogs(ctx, pod, logs)
			}
		}
	}

	return
}

func (c *JobExecutor) TailPodLogs(ctx context.Context, pod corev1.Pod, logs chan []byte) (err error) {
	count := int64(1)

	var containers []string
	for _, container := range pod.Spec.InitContainers {
		containers = append(containers, container.Name)
	}

	for _, container := range pod.Spec.Containers {
		containers = append(containers, container.Name)
	}

	go func() {
		defer close(logs)

		for _, container := range containers {
			podLogOptions := corev1.PodLogOptions{
				Follow:    true,
				TailLines: &count,
				Container: container,
			}

			podLogRequest := c.ClientSet.CoreV1().
				Pods(c.Namespace).
				GetLogs(pod.Name, &podLogOptions)

			stream, err := podLogRequest.Stream(ctx)
			if err != nil {
				c.Log.Errorw("stream error", "error", err)
				continue
			}

			reader := bufio.NewReader(stream)

			for {
				b, err := utils.ReadLongLine(reader)
				if err != nil {
					if err == io.EOF {
						err = nil
					}
					break
				}
				c.Log.Debug("TailPodLogs stream scan", "out", b, "pod", pod.Name)
				logs <- b
			}

			if err != nil {
				c.Log.Errorw("scanner error", "error", err)
			}
		}
	}()
	return
}

// GetPodLogError returns last line as error
func (c *JobExecutor) GetPodLogError(ctx context.Context, pod corev1.Pod) (logsBytes []byte, err error) {
	// error line should be last one
	return executor.GetPodLogs(ctx, c.ClientSet, c.Namespace, pod, 1)
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
	entry, err := output.GetLogEntry(errorLog)
	if err != nil {
		l.Errorw("GetLogEntry error", "error", err, "input", string(errorLog), "pod", pod)
		return errors.Errorf("GetLogEntry error: %v", err)
	}

	l.Infow("got last log entry", "log", entry.String())
	return errors.Errorf("error from last log entry: %s", entry.String())
}

// Abort aborts K8S by job name
func (c *JobExecutor) Abort(ctx context.Context, execution *testkube.Execution) (result *testkube.ExecutionResult, err error) {
	l := c.Log.With("execution", execution.Id)
	result, _ = executor.AbortJob(ctx, c.ClientSet, c.Namespace, execution.Id)
	l.Debugw("job aborted", "execution", execution.Id, "result", result)
	if err := c.stopExecution(ctx, l, execution, result, false, nil); err != nil {
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
	result = &testkube.ExecutionResult{
		Status: testkube.ExecutionStatusTimeout,
	}
	if err := c.stopExecution(ctx, l, &execution, result, false, nil); err != nil {
		l.Errorw("error stopping execution on job executor timeout", "error", err)
	}

	return
}

// NewJobSpec is a method to create new job spec
func NewJobSpec(log *zap.SugaredLogger, options JobOptions) (*batchv1.Job, error) {
	secretEnvVars := executor.PrepareSecretEnvs(options.SecretEnvs, options.Variables,
		options.UsernameSecret, options.TokenSecret)

	tmpl, err := template.New("job").Parse(options.JobTemplate)
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
		tmplExt, err := template.New("jobExt").Parse(options.JobTemplateExtensions)
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

	env := append(executor.RunnerEnvVars, secretEnvVars...)
	if options.HTTPProxy != "" {
		env = append(env, corev1.EnvVar{Name: "HTTP_PROXY", Value: options.HTTPProxy})
	}

	if options.HTTPSProxy != "" {
		env = append(env, corev1.EnvVar{Name: "HTTPS_PROXY", Value: options.HTTPSProxy})
	}

	for i := range job.Spec.Template.Spec.InitContainers {
		job.Spec.Template.Spec.InitContainers[i].Env = append(job.Spec.Template.Spec.InitContainers[i].Env, env...)
	}

	for i := range job.Spec.Template.Spec.Containers {
		job.Spec.Template.Spec.Containers[i].Env = append(job.Spec.Template.Spec.Containers[i].Env, env...)
		// override container image if provided
		if options.ImageOverride != "" {
			job.Spec.Template.Spec.Containers[i].Image = options.ImageOverride
		}
	}

	return &job, nil
}

func NewJobOptions(initImage, jobTemplate string, serviceAccountName string, execution testkube.Execution, options ExecuteOptions) (jobOptions JobOptions, err error) {
	jsn, err := json.Marshal(execution)
	if err != nil {
		return jobOptions, err
	}

	jobOptions = NewJobOptionsFromExecutionOptions(options)
	jobOptions.Name = execution.Id
	jobOptions.Namespace = execution.TestNamespace
	jobOptions.Jsn = string(jsn)
	jobOptions.InitImage = initImage
	jobOptions.TestName = execution.TestName
	if jobOptions.JobTemplate == "" {
		jobOptions.JobTemplate = jobTemplate
	}
	jobOptions.Variables = execution.Variables
	jobOptions.ImagePullSecrets = options.ImagePullSecretNames
	jobOptions.ServiceAccountName = serviceAccountName
	return
}
