package containerexecutor

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/pkg/featureflags"
	"github.com/kubeshop/testkube/pkg/imageinspector"
	"github.com/kubeshop/testkube/pkg/repository/config"
	"github.com/kubeshop/testkube/pkg/secret"
	"github.com/kubeshop/testkube/pkg/utils"

	"github.com/kubeshop/testkube/pkg/repository/result"

	"github.com/kubeshop/testkube/pkg/version"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"

	executorv1 "github.com/kubeshop/testkube-operator/api/executor/v1"
	executorsclientv1 "github.com/kubeshop/testkube-operator/pkg/client/executors/v1"
	templatesv1 "github.com/kubeshop/testkube-operator/pkg/client/templates/v1"
	testexecutionsv1 "github.com/kubeshop/testkube-operator/pkg/client/testexecutions/v1"
	testsv3 "github.com/kubeshop/testkube-operator/pkg/client/tests/v3"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executor"
	"github.com/kubeshop/testkube/pkg/executor/client"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/k8sclient"
	"github.com/kubeshop/testkube/pkg/log"
	logsclient "github.com/kubeshop/testkube/pkg/logs/client"
	testexecutionsmapper "github.com/kubeshop/testkube/pkg/mapper/testexecutions"
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
	images executor.Images,
	templates executor.Templates,
	imageInspector imageinspector.Inspector,
	serviceAccountNames map[string]string,
	metrics ExecutionMetric,
	emiter EventEmitter,
	configMap config.Repository,
	executorsClient executorsclientv1.Interface,
	testsClient testsv3.Interface,
	testExecutionsClient testexecutionsv1.Interface,
	templatesClient templatesv1.Interface,
	registry string,
	podStartTimeout time.Duration,
	clusterID string,
	dashboardURI string,
	apiURI string,
	natsUri string,
	debug bool,
	logsStream logsclient.Stream,
	features featureflags.FeatureFlags,
) (client *ContainerExecutor, err error) {
	clientSet, err := k8sclient.ConnectToK8s()
	if err != nil {
		return client, err
	}

	if serviceAccountNames == nil {
		serviceAccountNames = make(map[string]string)
	}

	return &ContainerExecutor{
		clientSet:            clientSet,
		repository:           repo,
		log:                  log.DefaultLogger,
		images:               images,
		templates:            templates,
		imageInspector:       imageInspector,
		configMap:            configMap,
		serviceAccountNames:  serviceAccountNames,
		metrics:              metrics,
		emitter:              emiter,
		testsClient:          testsClient,
		executorsClient:      executorsClient,
		testExecutionsClient: testExecutionsClient,
		templatesClient:      templatesClient,
		registry:             registry,
		podStartTimeout:      podStartTimeout,
		clusterID:            clusterID,
		dashboardURI:         dashboardURI,
		apiURI:               apiURI,
		natsURI:              natsUri,
		debug:                debug,
		logsStream:           logsStream,
		features:             features,
	}, nil
}

type ExecutionMetric interface {
	IncAndObserveExecuteTest(execution testkube.Execution, dashboardURI string)
}

// ContainerExecutor is container for managing job executor dependencies
type ContainerExecutor struct {
	repository           result.Repository
	log                  *zap.SugaredLogger
	clientSet            kubernetes.Interface
	images               executor.Images
	templates            executor.Templates
	imageInspector       imageinspector.Inspector
	metrics              ExecutionMetric
	emitter              EventEmitter
	configMap            config.Repository
	serviceAccountNames  map[string]string
	testsClient          testsv3.Interface
	executorsClient      executorsclientv1.Interface
	testExecutionsClient testexecutionsv1.Interface
	templatesClient      templatesv1.Interface
	registry             string
	podStartTimeout      time.Duration
	clusterID            string
	dashboardURI         string
	apiURI               string
	natsURI              string
	debug                bool
	logsStream           logsclient.Stream
	features             featureflags.FeatureFlags
}

type JobOptions struct {
	Name                      string
	Namespace                 string
	Image                     string
	ImagePullSecrets          []string
	Command                   []string
	Args                      []string
	WorkingDir                string
	Jsn                       string
	TestName                  string
	InitImage                 string
	ScraperImage              string
	JobTemplate               string
	ScraperTemplate           string
	PvcTemplate               string
	SecretEnvs                map[string]string
	Envs                      map[string]string
	HTTPProxy                 string
	HTTPSProxy                string
	UsernameSecret            *testkube.SecretRef
	TokenSecret               *testkube.SecretRef
	CertificateSecret         string
	AgentAPITLSSecret         string
	Variables                 map[string]testkube.Variable
	ActiveDeadlineSeconds     int64
	ArtifactRequest           *testkube.ArtifactRequest
	ServiceAccountName        string
	DelaySeconds              int
	JobTemplateExtensions     string
	ScraperTemplateExtensions string
	PvcTemplateExtensions     string
	EnvConfigMaps             []testkube.EnvReference
	EnvSecrets                []testkube.EnvReference
	Labels                    map[string]string
	Registry                  string
	ClusterID                 string
	ExecutionNumber           int32
	ContextType               string
	ContextData               string
	Debug                     bool
	LogSidecarImage           string
	NatsUri                   string
	APIURI                    string
	Features                  featureflags.FeatureFlags
}

// Logs returns job logs stream channel using kubernetes api
func (c *ContainerExecutor) Logs(ctx context.Context, id, namespace string) (out chan output.Output, err error) {
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
		if supportArtifacts && execution.ArtifactRequest != nil &&
			execution.ArtifactRequest.StorageClassName != "" {
			ids = append(ids, id+"-scraper")
		}

		for _, podName := range ids {
			logs := make(chan []byte)

			if err := c.TailJobLogs(ctx, podName, namespace, logs); err != nil {
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
		if cErr := c.cleanPVCVolume(ctx, execution); cErr != nil {
			c.log.Errorw("error cleaning pvc volume", "error", cErr)
		}

		return executionResult, err
	}

	podsClient := c.clientSet.CoreV1().Pods(execution.TestNamespace)
	pods, err := executor.GetJobPods(ctx, podsClient, execution.Id, 1, 10)
	if err != nil {
		executionResult.Err(err)
		if cErr := c.cleanPVCVolume(ctx, execution); cErr != nil {
			c.log.Errorw("error cleaning pvc volume", "error", cErr)
		}

		return executionResult, err
	}

	l := c.log.With("executionID", execution.Id, "sync", options.Sync)

	for _, pod := range pods.Items {
		if pod.Status.Phase != corev1.PodRunning && pod.Labels["job-name"] == execution.Id {
			if options.Sync {
				return c.updateResultsFromPod(ctx, pod, l, execution, jobOptions, options.Request.NegativeTest)
			}

			// async wait for complete status or error
			go func(pod corev1.Pod) {
				_, err := c.updateResultsFromPod(ctx, pod, l, execution, jobOptions, options.Request.NegativeTest)
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

// createJob creates new Kubernetes job based on execution and execute options
func (c *ContainerExecutor) createJob(ctx context.Context, execution testkube.Execution, options client.ExecuteOptions) (*JobOptions, error) {
	jobsClient := c.clientSet.BatchV1().Jobs(execution.TestNamespace)

	// Fallback to one-time inspector when non-default namespace is needed
	inspector := c.imageInspector
	if len(options.ImagePullSecretNames) > 0 && options.Namespace != "" && execution.TestNamespace != options.Namespace {
		secretClient, err := secret.NewClient(options.Namespace)
		if err != nil {
			return nil, errors.Wrap(err, "failed to build secrets client")
		}
		inspector = imageinspector.NewInspector(c.registry, imageinspector.NewSkopeoFetcher(), imageinspector.NewSecretFetcher(secretClient))
	}

	jobOptions, err := NewJobOptions(c.log, c.templatesClient, c.images, c.templates, inspector,
		c.serviceAccountNames, c.registry, c.clusterID, c.apiURI, execution, options, c.natsURI, c.debug)
	if err != nil {
		return nil, err
	}

	if jobOptions.ArtifactRequest != nil &&
		jobOptions.ArtifactRequest.StorageClassName != "" {
		c.log.Debug("creating persistent volume claim with options", "options", jobOptions)
		pvcsClient := c.clientSet.CoreV1().PersistentVolumeClaims(execution.TestNamespace)
		pvcSpec, err := client.NewPersistentVolumeClaimSpec(c.log, NewPVCOptionsFromJobOptions(*jobOptions))
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

func (c *ContainerExecutor) cleanPVCVolume(ctx context.Context, execution *testkube.Execution) error {
	if execution.ArtifactRequest != nil &&
		execution.ArtifactRequest.StorageClassName != "" {
		pvcsClient := c.clientSet.CoreV1().PersistentVolumeClaims(execution.TestNamespace)
		if err := pvcsClient.Delete(ctx, execution.Id+"-pvc", metav1.DeleteOptions{}); err != nil {
			return err
		}
	}

	return nil
}

// updateResultsFromPod watches logs and stores results if execution is finished
func (c *ContainerExecutor) updateResultsFromPod(
	ctx context.Context,
	executorPod corev1.Pod,
	l *zap.SugaredLogger,
	execution *testkube.Execution,
	jobOptions *JobOptions,
	isNegativeTest bool,
) (*testkube.ExecutionResult, error) {
	var err error

	// save stop time and final state
	defer func() {
		c.stopExecution(ctx, execution, execution.ExecutionResult, isNegativeTest)

		if err := c.cleanPVCVolume(ctx, execution); err != nil {
			l.Errorw("error cleaning pvc volume", "error", err)
		}
	}()

	// wait for pod
	l.Debug("poll immediate waiting for executor pod")
	if err = wait.PollUntilContextTimeout(ctx, pollInterval, c.podStartTimeout, true, executor.IsPodLoggable(c.clientSet, executorPod.Name, execution.TestNamespace)); err != nil {
		l.Errorw("waiting for executor pod started error", "error", err)
	} else if err = wait.PollUntilContextTimeout(ctx, pollInterval, pollTimeout, true, executor.IsPodReady(c.clientSet, executorPod.Name, execution.TestNamespace)); err != nil {
		// continue on poll err and try to get logs later
		l.Errorw("waiting for executor pod complete error", "error", err)
	}
	if err != nil {
		execution.ExecutionResult.Err(err)
	}
	l.Debug("poll executor immediate end")

	// we need to retrieve the Pod to get its latest status
	podsClient := c.clientSet.CoreV1().Pods(execution.TestNamespace)
	latestExecutorPod, err := podsClient.Get(context.Background(), executorPod.Name, metav1.GetOptions{})
	if err != nil {
		return execution.ExecutionResult, err
	}

	var scraperLogs []byte
	if jobOptions.ArtifactRequest != nil &&
		jobOptions.ArtifactRequest.StorageClassName != "" {
		c.log.Debug("creating scraper job with options", "options", jobOptions)
		jobsClient := c.clientSet.BatchV1().Jobs(execution.TestNamespace)
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
				if err = wait.PollUntilContextTimeout(ctx, pollInterval, c.podStartTimeout, true, executor.IsPodLoggable(c.clientSet, scraperPod.Name, execution.TestNamespace)); err != nil {
					l.Errorw("waiting for scraper pod started error", "error", err)
				} else if err = wait.PollUntilContextTimeout(ctx, pollInterval, pollTimeout, true, executor.IsPodReady(c.clientSet, scraperPod.Name, execution.TestNamespace)); err != nil {
					// continue on poll err and try to get logs later
					l.Errorw("waiting for scraper pod complete error", "error", err)
				}
				l.Debug("poll scraper immediate end")

				latestScraperPod, err := podsClient.Get(context.Background(), scraperPod.Name, metav1.GetOptions{})
				if err != nil {
					return execution.ExecutionResult, err
				}

				switch latestScraperPod.Status.Phase {
				case corev1.PodSucceeded:
					execution.ExecutionResult.Success()
				case corev1.PodFailed:
					execution.ExecutionResult.Error()
				}

				scraperLogs, err = executor.GetPodLogs(ctx, c.clientSet, execution.TestNamespace, *latestScraperPod)
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

	executorLogs, err := executor.GetPodLogs(ctx, c.clientSet, execution.TestNamespace, *latestExecutorPod)
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

	// parse container output log (mixed JSON and plain text stream)
	executionResult, output, err := output.ParseContainerOutput(executorLogs)
	if err != nil {
		l.Errorw("parse output error", "error", err)
		execution.ExecutionResult.Output = output
		execution.ExecutionResult.Err(err)
		err = c.repository.UpdateResult(ctx, execution.Id, *execution)
		if err != nil {
			l.Infow("Update result", "error", err)
		}
		return execution.ExecutionResult, err
	}

	if executionResult != nil {
		execution.ExecutionResult = executionResult
	}

	// don't attach logs if logs v2 is enabled - they will be streamed through the logs service
	attachLogs := !c.features.LogsV2
	if attachLogs {
		execution.ExecutionResult.Output = output
	}

	if execution.ExecutionResult.IsFailed() {
		errorMessage := execution.ExecutionResult.ErrorMessage
		if errorMessage == "" {
			errorMessage = executor.GetPodErrorMessage(ctx, c.clientSet, latestExecutorPod)
		}

		execution.ExecutionResult.ErrorMessage = errorMessage
	}

	l.Infow("container execution completed saving result", "executionId", execution.Id, "status", execution.ExecutionResult.Status)
	err = c.repository.UpdateResult(ctx, execution.Id, *execution)
	if err != nil {
		l.Errorw("Update execution result error", "error", err)
	}
	return execution.ExecutionResult, nil
}

func (c *ContainerExecutor) stopExecution(ctx context.Context,
	execution *testkube.Execution,
	result *testkube.ExecutionResult,
	isNegativeTest bool,
) {
	c.log.Debugw("stopping execution", "isNegativeTest", isNegativeTest, "test", execution.TestName)
	execution.Stop()

	if isNegativeTest {
		if result.IsFailed() {
			c.log.Debugw("test run was expected to fail, and it failed as expected", "test", execution.TestName)
			execution.ExecutionResult.Status = testkube.ExecutionStatusPassed
			result.Status = testkube.ExecutionStatusPassed
			result.Output = result.Output + "\nTest run was expected to fail, and it failed as expected"
		} else {
			c.log.Debugw("test run was expected to fail - the result will be reversed", "test", execution.TestName)
			execution.ExecutionResult.Status = testkube.ExecutionStatusFailed
			result.Status = testkube.ExecutionStatusFailed
			result.Output = result.Output + "\nTest run was expected to fail, the result will be reversed"
		}

		err := c.repository.UpdateResult(ctx, execution.Id, *execution)
		if err != nil {
			c.log.Errorw("Update execution result error", "error", err)
		}
	}

	err := c.repository.EndExecution(ctx, *execution)
	if err != nil {
		c.log.Errorw("Update execution result error", "error", err)
	}

	// metrics increase
	execution.ExecutionResult = result
	c.metrics.IncAndObserveExecuteTest(*execution, c.dashboardURI)

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

	if execution.TestExecutionName != "" {
		testExecution, err := c.testExecutionsClient.Get(execution.TestExecutionName)
		if err != nil {
			c.log.Errorw("getting test execution error", "error", err)
		}

		if testExecution != nil {
			testExecution.Status = testexecutionsmapper.MapAPIToCRD(execution, testExecution.Generation)
			if err = c.testExecutionsClient.UpdateStatus(testExecution); err != nil {
				c.log.Errorw("updating test execution error", "error", err)
			}
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
	// for image, HTTP request takes priority, then test spec, then executor
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

	return &JobOptions{
		Image:                     image,
		ImagePullSecrets:          options.ImagePullSecretNames,
		Args:                      options.Request.Args,
		Command:                   options.Request.Command,
		WorkingDir:                workingDir,
		TestName:                  options.TestName,
		Namespace:                 options.Namespace,
		Envs:                      options.Request.Envs,
		SecretEnvs:                options.Request.SecretEnvs,
		HTTPProxy:                 options.Request.HttpProxy,
		HTTPSProxy:                options.Request.HttpsProxy,
		UsernameSecret:            options.UsernameSecret,
		TokenSecret:               options.TokenSecret,
		CertificateSecret:         options.CertificateSecret,
		AgentAPITLSSecret:         options.AgentAPITLSSecret,
		ActiveDeadlineSeconds:     options.Request.ActiveDeadlineSeconds,
		ArtifactRequest:           artifactRequest,
		DelaySeconds:              jobDelaySeconds,
		JobTemplate:               options.ExecutorSpec.JobTemplate,
		JobTemplateExtensions:     options.Request.JobTemplate,
		ScraperTemplateExtensions: options.Request.ScraperTemplate,
		PvcTemplateExtensions:     options.Request.PvcTemplate,
		EnvConfigMaps:             options.Request.EnvConfigMaps,
		EnvSecrets:                options.Request.EnvSecrets,
		Labels:                    labels,
		ExecutionNumber:           options.Request.Number,
		ContextType:               contextType,
		ContextData:               contextData,
		Features:                  options.Features,
	}
}

// Abort K8sJob aborts K8S by job name
func (c *ContainerExecutor) Abort(ctx context.Context, execution *testkube.Execution) (*testkube.ExecutionResult, error) {
	return executor.AbortJob(ctx, c.clientSet, execution.TestNamespace, execution.Id)
}

func NewPVCOptionsFromJobOptions(options JobOptions) client.PVCOptions {
	return client.PVCOptions{
		Name:                  options.Name,
		Namespace:             options.Namespace,
		PvcTemplate:           options.PvcTemplate,
		PvcTemplateExtensions: options.PvcTemplateExtensions,
		ArtifactRequest:       options.ArtifactRequest,
	}
}
