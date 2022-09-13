package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/template"
	"time"

	"go.uber.org/zap"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	tcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/kubeshop/testkube/internal/pkg/api/repository/result"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event"
	"github.com/kubeshop/testkube/pkg/executor/output"
	secretenv "github.com/kubeshop/testkube/pkg/executor/secret"
	"github.com/kubeshop/testkube/pkg/k8sclient"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/secret"
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
)

var (
	envVars = []corev1.EnvVar{
		{
			Name:  "DEBUG",
			Value: os.Getenv("DEBUG"),
		},
		{
			Name:  "RUNNER_ENDPOINT",
			Value: os.Getenv("STORAGE_ENDPOINT"),
		},
		{
			Name:  "RUNNER_ACCESSKEYID",
			Value: os.Getenv("STORAGE_ACCESSKEYID"),
		},
		{
			Name:  "RUNNER_SECRETACCESSKEY",
			Value: os.Getenv("STORAGE_SECRETACCESSKEY"),
		},
		{
			Name:  "RUNNER_LOCATION",
			Value: os.Getenv("STORAGE_LOCATION"),
		},
		{
			Name:  "RUNNER_TOKEN",
			Value: os.Getenv("STORAGE_TOKEN"),
		},
		{
			Name:  "RUNNER_SSL",
			Value: os.Getenv("STORAGE_SSL"),
		},
		{
			Name:  "RUNNER_SCRAPPERENABLED",
			Value: os.Getenv("SCRAPPERENABLED"),
		},
		{
			Name:  "RUNNER_DATADIR",
			Value: volumeDir,
		},
	}
)

// NewJobExecutor creates new job executor
func NewJobExecutor(repo result.Repository, namespace, initImage, jobTemplate string, metrics ExecutionCounter, emiter *event.Emitter) (client *JobExecutor, err error) {
	clientSet, err := k8sclient.ConnectToK8s()
	if err != nil {
		return client, err
	}

	return &JobExecutor{
		ClientSet:   clientSet,
		Repository:  repo,
		Log:         log.DefaultLogger,
		Namespace:   namespace,
		initImage:   initImage,
		jobTemplate: jobTemplate,
		metrics:     metrics,
		Emitter:     emiter,
	}, nil
}

type ExecutionCounter interface {
	IncExecuteTest(execution testkube.Execution)
}

// JobExecutor is container for managing job executor dependencies
type JobExecutor struct {
	Repository  result.Repository
	Log         *zap.SugaredLogger
	ClientSet   *kubernetes.Clientset
	Namespace   string
	Cmd         string
	initImage   string
	jobTemplate string
	metrics     ExecutionCounter
	Emitter     *event.Emitter
}

type JobOptions struct {
	Name           string
	Namespace      string
	Image          string
	ImageOverride  string
	Jsn            string
	TestName       string
	InitImage      string
	JobTemplate    string
	HasSecrets     bool
	SecretEnvs     map[string]string
	HTTPProxy      string
	HTTPSProxy     string
	UsernameSecret *testkube.SecretRef
	TokenSecret    *testkube.SecretRef
	Variables      map[string]testkube.Variable
}

// Logs returns job logs stream channel using kubernetes api
func (c JobExecutor) Logs(id string) (out chan output.Output, err error) {
	out = make(chan output.Output)
	logs := make(chan []byte)

	go func() {
		defer func() {
			c.Log.Debug("closing JobExecutor.Logs out log")
			close(out)
		}()

		if err := c.TailJobLogs(id, logs); err != nil {
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
func (c JobExecutor) Execute(execution *testkube.Execution, options ExecuteOptions) (result testkube.ExecutionResult, err error) {

	result = testkube.NewRunningExecutionResult()

	ctx := context.Background()
	err = c.CreateJob(ctx, *execution, options)
	if err != nil {
		return result.Err(err), err
	}

	podsClient := c.ClientSet.CoreV1().Pods(c.Namespace)
	pods, err := c.GetJobPods(podsClient, execution.Id, 1, 10)
	if err != nil {
		return result.Err(err), err
	}

	l := c.Log.With("executionID", execution.Id, "type", "async")

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
func (c JobExecutor) ExecuteSync(execution *testkube.Execution, options ExecuteOptions) (result testkube.ExecutionResult, err error) {
	result = testkube.NewRunningExecutionResult()

	ctx := context.Background()
	err = c.CreateJob(ctx, *execution, options)
	if err != nil {
		return result.Err(err), err
	}

	podsClient := c.ClientSet.CoreV1().Pods(c.Namespace)
	pods, err := c.GetJobPods(podsClient, execution.Id, 1, 10)
	if err != nil {
		return result.Err(err), err
	}

	l := c.Log.With("executionID", execution.Id, "type", "sync")

	// get job pod and
	for _, pod := range pods.Items {
		if pod.Status.Phase != corev1.PodRunning && pod.Labels["job-name"] == execution.Id {
			return c.updateResultsFromPod(ctx, pod, l, execution, result)
		}
	}

	l.Debugw("no pods was found", "totalPodsCount", len(pods.Items))

	return

}

// CreateJob creates new Kubernetes job based on execution and execute options
func (c JobExecutor) CreateJob(ctx context.Context, execution testkube.Execution, options ExecuteOptions) error {
	jobs := c.ClientSet.BatchV1().Jobs(c.Namespace)

	jobOptions, err := NewJobOptions(c.initImage, c.jobTemplate, execution, options)
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
func (c JobExecutor) updateResultsFromPod(ctx context.Context, pod corev1.Pod, l *zap.SugaredLogger, execution *testkube.Execution, result testkube.ExecutionResult) (testkube.ExecutionResult, error) {
	var err error

	// save stop time and final state
	defer c.stopExecution(ctx, l, execution, &result)

	// wait for complete
	l.Debug("poll immediate waiting for pod to succeed")
	if err = wait.PollImmediate(pollInterval, pollTimeout, IsPodReady(c.ClientSet, pod.Name, c.Namespace)); err != nil {
		// continue on poll err and try to get logs later
		l.Errorw("waiting for pod complete error", "error", err)
	}
	l.Debug("poll immediate end")

	var logs []byte
	logs, err = c.GetPodLogs(pod)
	if err != nil {
		l.Errorw("get pod logs error", "error", err)
		err = c.Repository.UpdateResult(ctx, execution.Id, result.Err(err))
		if err != nil {
			l.Infow("Update result", "error", err)
		}
		return result, err
	}

	// parse job ouput log (JSON stream)
	result, _, err = output.ParseRunnerOutput(logs)
	if err != nil {
		l.Errorw("parse ouput error", "error", err)
		err = c.Repository.UpdateResult(ctx, execution.Id, result.Err(err))
		if err != nil {
			l.Errorw("Update execution result error", "error", err)
		}
		return result, err
	}

	l.Infow("execution completed saving result", "executionId", execution.Id, "status", result.Status)
	err = c.Repository.UpdateResult(ctx, execution.Id, result)
	if err != nil {
		l.Errorw("Update execution result error", "error", err)
	}
	return result, nil

}

func (c JobExecutor) stopExecution(ctx context.Context, l *zap.SugaredLogger, execution *testkube.Execution, result *testkube.ExecutionResult) {
	l.Debug("stopping execution")
	execution.Stop()
	err := c.Repository.EndExecution(ctx, execution.Id, execution.EndTime, execution.CalculateDuration())
	if err != nil {
		l.Errorw("Update execution result error", "error", err)
	}

	// metrics increase
	execution.ExecutionResult = result
	c.metrics.IncExecuteTest(*execution)

	c.Emitter.Notify(testkube.NewEventEndTestSuccess(execution))
}

// NewJobOptionsFromExecutionOptions compose JobOptions based on ExecuteOptions
func NewJobOptionsFromExecutionOptions(options ExecuteOptions) JobOptions {
	return JobOptions{
		Image:          options.ExecutorSpec.Image,
		ImageOverride:  options.ImageOverride,
		HasSecrets:     options.HasSecrets,
		JobTemplate:    options.ExecutorSpec.JobTemplate,
		TestName:       options.TestName,
		Namespace:      options.Namespace,
		SecretEnvs:     options.Request.SecretEnvs,
		HTTPProxy:      options.Request.HttpProxy,
		HTTPSProxy:     options.Request.HttpsProxy,
		UsernameSecret: options.UsernameSecret,
		TokenSecret:    options.TokenSecret,
	}
}

// GetJobPods returns job pods
func (c *JobExecutor) GetJobPods(podsClient tcorev1.PodInterface, jobName string, retryNr, retryCount int) (*corev1.PodList, error) {
	pods, err := podsClient.List(context.TODO(), metav1.ListOptions{LabelSelector: "job-name=" + jobName})
	if err != nil {
		return nil, err
	}
	if retryNr == retryCount {
		return nil, fmt.Errorf("retry count exceeeded, there are no active pods with given id=%s", jobName)
	}
	if len(pods.Items) == 0 {
		time.Sleep(time.Duration(retryNr * 500 * int(time.Millisecond))) // increase backoff timeout
		return c.GetJobPods(podsClient, jobName, retryNr+1, retryCount)
	}
	return pods, nil
}

// TailJobLogs - locates logs for job pod(s)
func (c *JobExecutor) TailJobLogs(id string, logs chan []byte) (err error) {

	podsClient := c.ClientSet.CoreV1().Pods(c.Namespace)
	ctx := context.Background()

	pods, err := c.GetJobPods(podsClient, id, 1, 10)
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
				err := fmt.Errorf("can't get pod logs, pod failed: %s/%s", pod.Namespace, pod.Name)
				l.Errorw(err.Error())
				return c.GetLastLogLineError(ctx, pod)

			default:
				l.Debugw("tailing job logs: waiting for pod to be ready")
				if err = wait.PollImmediate(pollInterval, pollTimeout, IsPodReady(c.ClientSet, pod.Name, c.Namespace)); err != nil {
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
				b, err := reader.ReadBytes('\n')
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
	return c.GetPodLogs(pod, 1)
}

// GetLastLogLineError return error if last line is failed
func (c *JobExecutor) GetLastLogLineError(ctx context.Context, pod corev1.Pod) error {
	l := c.Log.With("pod", pod.Name, "namespace", pod.Namespace)
	log, err := c.GetPodLogError(ctx, pod)
	if err != nil {
		return fmt.Errorf("getPodLogs error: %w", err)
	}

	l.Debugw("log", "got last log bytes", string(log)) // in case distorted log bytes
	entry, err := output.GetLogEntry(log)
	if err != nil {
		return fmt.Errorf("GetLogEntry error: %w", err)
	}

	c.Log.Errorw("got last log entry", "log", entry.String())
	return fmt.Errorf("error from last log entry: %s", entry.String())
}

// GetPodLogs returns pod logs bytes
func (c *JobExecutor) GetPodLogs(pod corev1.Pod, logLinesCount ...int64) (logs []byte, err error) {
	count := int64(100)
	if len(logLinesCount) > 0 {
		count = logLinesCount[0]
	}

	var containers []string
	for _, container := range pod.Spec.InitContainers {
		containers = append(containers, container.Name)
	}

	for _, container := range pod.Spec.Containers {
		containers = append(containers, container.Name)
	}

	for _, container := range containers {
		podLogOptions := corev1.PodLogOptions{
			Follow:    false,
			TailLines: &count,
			Container: container,
		}

		podLogRequest := c.ClientSet.CoreV1().
			Pods(c.Namespace).
			GetLogs(pod.Name, &podLogOptions)

		stream, err := podLogRequest.Stream(context.TODO())
		if err != nil {
			if len(logs) != 0 && strings.Contains(err.Error(), "PodInitializing") {
				return logs, nil
			}

			return logs, err
		}

		defer stream.Close()

		buf := new(bytes.Buffer)
		_, err = io.Copy(buf, stream)
		if err != nil {
			if len(logs) != 0 && strings.Contains(err.Error(), "PodInitializing") {
				return logs, nil
			}

			return logs, err
		}

		logs = append(logs, buf.Bytes()...)
	}

	return logs, nil
}

// AbortK8sJob aborts K8S by job name
func (c *JobExecutor) Abort(jobName string) *testkube.ExecutionResult {
	var zero int64 = 0
	bg := metav1.DeletePropagationBackground
	jobs := c.ClientSet.BatchV1().Jobs(c.Namespace)
	err := jobs.Delete(context.TODO(), jobName, metav1.DeleteOptions{
		GracePeriodSeconds: &zero,
		PropagationPolicy:  &bg,
	})
	if err != nil {
		return &testkube.ExecutionResult{
			Status: testkube.ExecutionStatusFailed,
			Output: err.Error(),
		}
	}
	return &testkube.ExecutionResult{
		Status: testkube.ExecutionStatusCancelled,
	}
}

// NewJobSpec is a method to create new job spec
func NewJobSpec(log *zap.SugaredLogger, options JobOptions) (*batchv1.Job, error) {
	secretEnvVars := prepareSecretEnvs(options)
	tmpl, err := template.New("job").Parse(options.JobTemplate)
	if err != nil {
		return nil, fmt.Errorf("creating job spec from options.JobTemplate error: %w", err)
	}

	options.Jsn = strings.ReplaceAll(options.Jsn, "'", "''")
	var buffer bytes.Buffer
	if err = tmpl.ExecuteTemplate(&buffer, "job", options); err != nil {
		return nil, fmt.Errorf("executing job spec template: %w", err)
	}

	var job batchv1.Job
	jobSpec := buffer.String()
	log.Debug("Job specification", jobSpec)
	decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewBufferString(jobSpec), len(jobSpec))
	if err := decoder.Decode(&job); err != nil {
		return nil, fmt.Errorf("decoding job spec error: %w", err)
	}

	env := append(envVars, secretEnvVars...)
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

// IsPodReady defines if pod is ready or failed for logs scrapping
func IsPodReady(c *kubernetes.Clientset, podName, namespace string) wait.ConditionFunc {
	return func() (bool, error) {
		pod, err := c.CoreV1().Pods(namespace).Get(context.Background(), podName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		switch pod.Status.Phase {
		case corev1.PodSucceeded:
			return true, nil
		case corev1.PodFailed:
			return true, fmt.Errorf("pod %s/%s failed", pod.Namespace, pod.Name)
		}
		return false, nil
	}
}

func NewJobOptions(initImage, jobTemplate string, execution testkube.Execution, options ExecuteOptions) (jobOptions JobOptions, err error) {
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

	return
}

// prepareSecetEnvs generates secret envs from job options
func prepareSecretEnvs(options JobOptions) (secretEnvVars []corev1.EnvVar) {
	secretEnvVars = secretenv.NewEnvManager().Prepare(options.SecretEnvs, options.Variables)

	// prepare git credentials
	var setSecrets bool
	var data = []struct {
		envVar    string
		secretRef *testkube.SecretRef
	}{
		{
			GitUsernameEnvVarName,
			options.UsernameSecret,
		},
		{
			GitTokenEnvVarName,
			options.TokenSecret,
		},
	}

	for _, value := range data {
		if value.secretRef != nil {
			setSecrets = true
			secretEnvVars = append(secretEnvVars, corev1.EnvVar{
				Name: value.envVar,
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: value.secretRef.Name,
						},
						Key: value.secretRef.Key,
					},
				},
			})
		}
	}

	if options.HasSecrets && !setSecrets {
		var data = []struct {
			envVar    string
			secretKey string
		}{
			{
				GitUsernameEnvVarName,
				GitUsernameSecretName,
			},
			{
				GitTokenEnvVarName,
				GitTokenSecretName,
			},
		}

		for _, value := range data {
			secretEnvVars = append(secretEnvVars, corev1.EnvVar{
				Name: value.envVar,
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: secret.GetMetadataName(options.TestName),
						},
						Key: value.secretKey,
					},
				},
			})
		}
	}

	return secretEnvVars
}
