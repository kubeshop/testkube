package options

import (
	"encoding/json"
	"path/filepath"
	"time"

	_ "embed"

	"github.com/kubeshop/testkube/internal/featureflags"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executor"
	"github.com/kubeshop/testkube/pkg/secret"
	"github.com/kubeshop/testkube/pkg/skopeo"
	"github.com/kubeshop/testkube/pkg/utils"

	executorv1 "github.com/kubeshop/testkube-operator/api/executor/v1"
	templatesv1 "github.com/kubeshop/testkube-operator/pkg/client/templates/v1"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
)

const (
	// EntrypointScriptName is entrypoint script name
	EntrypointScriptName    = "entrypoint.sh"
	pollTimeout             = 24 * time.Hour
	pollInterval            = 200 * time.Millisecond
	jobDefaultDelaySeconds  = 180
	jobArtifactDelaySeconds = 90
	repoPath                = "/data/repo"
)

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

	// job executor only features
	SlavePodTemplate string
}

// job executor
func NewJobOptions(
	log *zap.SugaredLogger,
	templatesClient templatesv1.Interface,
	images executor.Images,
	templates executor.Templates,
	serviceAccountName,
	registry, clusterID, apiURI string,
	execution testkube.Execution,
	executeOptions ExecuteOptions,
	natsURI string,
	debug bool,
) (opts JobOptions, err error) {
	opts = fromExecuteOptions(executeOptions)

	// options needed for Log sidecar
	if opts.Features.LogsV2 {
		opts.Debug = debug
		opts.NatsUri = natsURI
		opts.LogSidecarImage = images.LogSidecar
	}

	// append execution data to job options
	opts.Name = execution.Id
	opts.TestName = execution.TestName
	opts.Namespace = execution.TestNamespace
	opts.Variables = execution.Variables
	opts.ArtifactRequest = execution.ArtifactRequest

	// append additional data
	opts.ServiceAccountName = serviceAccountName
	opts.Registry = registry
	opts.ClusterID = clusterID
	opts.APIURI = apiURI

	// Init image
	opts.InitImage = images.Init
	if images.Scraper != "" {
		opts.ScraperImage = images.Scraper
	}

	// Pre/Post run scripts - this one was only in container executor
	if execution.PreRunScript != "" || execution.PostRunScript != "" {
		opts.Command = []string{filepath.Join(executor.VolumeDir, EntrypointScriptName)}
		if opts.Image != "" {
			cmd, shell, err := InspectDockerImage(opts.Namespace, registry, opts.Image, opts.ImagePullSecrets)
			if err == nil {
				if len(execution.Command) == 0 {
					execution.Command = cmd
				}

				execution.ContainerShell = shell
			} else {
				log.Errorw("Docker image inspection error", "error", err)
			}
		}
	}

	// Store execution data as JSON in string
	jsn, err := json.Marshal(execution)
	if err != nil {
		return opts, err
	}
	opts.Jsn = string(jsn)

	// Job template overrides
	if opts.JobTemplate == "" {
		opts.JobTemplate = templates.Job
	}
	if executeOptions.ExecutorSpec.JobTemplateReference != "" {
		template, err := templatesClient.Get(executeOptions.ExecutorSpec.JobTemplateReference)
		if err != nil {
			return opts, err
		}

		if template.Spec.Type_ != nil && (testkube.TemplateType(*template.Spec.Type_) == testkube.JOB_TemplateType || testkube.TemplateType(*template.Spec.Type_) == testkube.CONTAINER_TemplateType) {
			opts.JobTemplate = template.Spec.Body
		} else {
			log.Warnw("Not matched template type", "template", executeOptions.ExecutorSpec.JobTemplateReference)
		}
	}
	if executeOptions.Request.JobTemplateReference != "" {
		template, err := templatesClient.Get(executeOptions.Request.JobTemplateReference)
		if err != nil {
			return opts, err
		}

		if template.Spec.Type_ != nil && (testkube.TemplateType(*template.Spec.Type_) == testkube.JOB_TemplateType || testkube.TemplateType(*template.Spec.Type_) == testkube.CONTAINER_TemplateType) {
			opts.JobTemplate = template.Spec.Body
		} else {
			log.Warnw("Not matched template type", "template", executeOptions.Request.JobTemplateReference)
		}
	}

	// Scraper template overrides
	if templates.Scraper != "" {
		opts.ScraperTemplate = templates.Scraper
	}
	if executeOptions.Request.ScraperTemplateReference != "" {
		template, err := templatesClient.Get(executeOptions.Request.ScraperTemplateReference)
		if err != nil {
			return opts, err
		}

		if template.Spec.Type_ != nil && testkube.TemplateType(*template.Spec.Type_) == testkube.SCRAPER_TemplateType {
			opts.ScraperTemplate = template.Spec.Body
		} else {
			log.Warnw("Not matched template type", "template", executeOptions.Request.ScraperTemplateReference)
		}
	}

	// PVC template overrides
	if templates.PVC != "" {
		opts.PvcTemplate = templates.PVC
	}
	if executeOptions.Request.PvcTemplateReference != "" {
		template, err := templatesClient.Get(executeOptions.Request.PvcTemplateReference)
		if err != nil {
			return opts, err
		}

		if template.Spec.Type_ != nil && testkube.TemplateType(*template.Spec.Type_) == testkube.PVC_TemplateType {
			opts.PvcTemplate = template.Spec.Body
		} else {
			log.Warnw("Not matched template type", "template", executeOptions.Request.PvcTemplateReference)
		}
	}

	// Slave pod template overrides
	opts.SlavePodTemplate = templates.SlavePod
	if executeOptions.Request.SlavePodRequest != nil && executeOptions.Request.SlavePodRequest.PodTemplateReference != "" {
		template, err := templatesClient.Get(executeOptions.Request.SlavePodRequest.PodTemplateReference)
		if err != nil {
			return opts, err
		}

		if template.Spec.Type_ != nil && testkube.TemplateType(*template.Spec.Type_) == testkube.POD_TemplateType {
			opts.SlavePodTemplate = template.Spec.Body
		} else {
			log.Warnw("Not matched template type", "template", executeOptions.Request.SlavePodRequest.PodTemplateReference)
		}
	}

	if executeOptions.ExecutorSpec.Slaves != nil {
		cfg := executor.GetSlavesConfigs(
			images.Init,
			*executeOptions.ExecutorSpec.Slaves,
			opts.Registry,
			opts.ServiceAccountName,
			opts.CertificateSecret,
			opts.SlavePodTemplate,
			opts.ImagePullSecrets,
			opts.EnvConfigMaps,
			opts.EnvSecrets,
			int(opts.ActiveDeadlineSeconds),
		)
		slvesConfigs, err := json.Marshal(cfg)

		if err != nil {
			return opts, err
		}

		opts.Variables[executor.SlavesConfigsEnv] = testkube.NewBasicVariable(executor.SlavesConfigsEnv, string(slvesConfigs))
	}

	return
}

// job
// fromExecuteOptions compose JobOptions based on ExecuteOptions
func fromExecuteOptions(options ExecuteOptions) JobOptions {

	// for args, command and image, HTTP request takes priority, then test spec, then executor
	var args []string
	argsMode := options.Request.ArgsMode
	if options.TestSpec.ExecutionRequest != nil && argsMode == "" {
		argsMode = string(options.TestSpec.ExecutionRequest.ArgsMode)
	}

	if argsMode == string(testkube.ArgsModeTypeAppend) || argsMode == "" {
		args = options.Request.Args
		if options.TestSpec.ExecutionRequest != nil && len(args) == 0 {
			args = options.TestSpec.ExecutionRequest.Args
		}

		args = append(options.ExecutorSpec.Args, args...)
	}

	if argsMode == string(testkube.ArgsModeTypeOverride) {
		args = options.Request.Args
		if options.TestSpec.ExecutionRequest != nil && len(args) == 0 {
			args = options.TestSpec.ExecutionRequest.Args
		}
	}

	var command []string
	if len(options.ExecutorSpec.Command) != 0 {
		command = options.ExecutorSpec.Command
	}

	if options.TestSpec.ExecutionRequest != nil &&
		len(options.TestSpec.ExecutionRequest.Command) != 0 {
		command = options.TestSpec.ExecutionRequest.Command
	}

	if len(options.Request.Command) != 0 {
		command = options.Request.Command
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

	// TODO this one is from container executor - confirm if we can go with it in job executor
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

	opts := JobOptions{
		Image:                     image,
		ImagePullSecrets:          options.ImagePullSecretNames,
		Args:                      args,
		Command:                   command,
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

	return opts
}

// InspectDockerImage inspects docker image
func InspectDockerImage(namespace, registry, image string, imageSecrets []string) ([]string, string, error) {
	inspector := skopeo.NewClient()
	if len(imageSecrets) != 0 {
		secretClient, err := secret.NewClient(namespace)
		if err != nil {
			return nil, "", err
		}

		var secrets []corev1.Secret
		for _, imageSecret := range imageSecrets {
			object, err := secretClient.GetObject(imageSecret)
			if err != nil {
				return nil, "", err
			}

			secrets = append(secrets, *object)
		}

		inspector, err = skopeo.NewClientFromSecrets(secrets, registry)
		if err != nil {
			return nil, "", err
		}
	}

	dockerImage, err := inspector.Inspect(image)
	if err != nil {
		return nil, "", err
	}

	return append(dockerImage.Config.Entrypoint, dockerImage.Config.Cmd...), dockerImage.Shell, nil
}
