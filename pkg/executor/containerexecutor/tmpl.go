package containerexecutor

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
	"sigs.k8s.io/kustomize/kyaml/yaml/merge2"

	templatesv1 "github.com/kubeshop/testkube-operator/pkg/client/templates/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executor"
	"github.com/kubeshop/testkube/pkg/executor/client"
	"github.com/kubeshop/testkube/pkg/executor/env"
	"github.com/kubeshop/testkube/pkg/imageinspector"
	"github.com/kubeshop/testkube/pkg/utils"
)

const (
	// EntrypointScriptName is entrypoint script name
	EntrypointScriptName = "entrypoint.sh"
)

//go:embed templates/job.tmpl
var defaultJobTemplate string

// NewExecutorJobSpec is a method to create new executor job spec
func NewExecutorJobSpec(log *zap.SugaredLogger, options *JobOptions) (*batchv1.Job, error) {
	envManager := env.NewManager()
	secretEnvVars := append(envManager.PrepareSecrets(options.SecretEnvs, options.Variables),
		envManager.PrepareGitCredentials(options.UsernameSecret, options.TokenSecret)...)

	tmpl, err := utils.NewTemplate("job").Parse(options.JobTemplate)
	if err != nil {
		return nil, fmt.Errorf("creating job spec from executor template error: %w", err)
	}

	options.Jsn = strings.ReplaceAll(options.Jsn, "'", "''")
	for i := range options.Command {
		if options.Command[i] != "" {
			options.Command[i] = fmt.Sprintf("%q", options.Command[i])
		}
	}

	for i := range options.Args {
		if options.Args[i] != "" {
			options.Args[i] = fmt.Sprintf("%q", options.Args[i])
		}
	}

	var buffer bytes.Buffer
	if err = tmpl.ExecuteTemplate(&buffer, "job", options); err != nil {
		return nil, fmt.Errorf("executing job spec executor template: %w", err)
	}

	var job batchv1.Job
	jobSpec := buffer.String()
	if options.JobTemplateExtensions != "" {
		tmplExt, err := utils.NewTemplate("jobExt").Parse(options.JobTemplateExtensions)
		if err != nil {
			return nil, fmt.Errorf("creating job extensions spec from executor template error: %w", err)
		}

		var bufferExt bytes.Buffer
		if err = tmplExt.ExecuteTemplate(&bufferExt, "jobExt", options); err != nil {
			return nil, fmt.Errorf("executing job extensions spec executor template: %w", err)
		}

		if jobSpec, err = merge2.MergeStrings(bufferExt.String(), jobSpec, false, kyaml.MergeOptions{}); err != nil {
			return nil, fmt.Errorf("merging job spec executor templates: %w", err)
		}
	}

	decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewBufferString(jobSpec), len(jobSpec))
	if err := decoder.Decode(&job); err != nil {
		return nil, fmt.Errorf("decoding executor job spec error: %w", err)
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

	// envs needed for logs sidecar
	if options.Features.LogsV2 {
		envs = append(envs, corev1.EnvVar{Name: "ID", Value: options.Name})
		envs = append(envs, corev1.EnvVar{Name: "NATS_URI", Value: options.NatsUri})
		envs = append(envs, corev1.EnvVar{Name: "NAMESPACE", Value: options.Namespace})
	}

	for i := range job.Spec.Template.Spec.InitContainers {
		job.Spec.Template.Spec.InitContainers[i].Env = append(job.Spec.Template.Spec.InitContainers[i].Env, envs...)
	}

	for i := range job.Spec.Template.Spec.Containers {
		job.Spec.Template.Spec.Containers[i].Env = append(job.Spec.Template.Spec.Containers[i].Env, envs...)
	}

	return &job, nil
}

// NewScraperJobSpec is a method to create new scraper job spec
func NewScraperJobSpec(log *zap.SugaredLogger, options *JobOptions) (*batchv1.Job, error) {
	tmpl, err := utils.NewTemplate("job").Parse(options.ScraperTemplate)
	if err != nil {
		return nil, fmt.Errorf("creating job spec from scraper template error: %w", err)
	}

	options.Jsn = strings.ReplaceAll(options.Jsn, "'", "''")
	var buffer bytes.Buffer
	if err = tmpl.ExecuteTemplate(&buffer, "job", options); err != nil {
		return nil, fmt.Errorf("executing job spec scraper template: %w", err)
	}

	var job batchv1.Job
	jobSpec := buffer.String()
	if options.ScraperTemplateExtensions != "" {
		tmplExt, err := utils.NewTemplate("jobExt").Parse(options.ScraperTemplateExtensions)
		if err != nil {
			return nil, fmt.Errorf("creating scraper extensions spec from executor template error: %w", err)
		}

		var bufferExt bytes.Buffer
		if err = tmplExt.ExecuteTemplate(&bufferExt, "jobExt", options); err != nil {
			return nil, fmt.Errorf("executing scraper extensions spec executor template: %w", err)
		}

		if jobSpec, err = merge2.MergeStrings(bufferExt.String(), jobSpec, false, kyaml.MergeOptions{}); err != nil {
			return nil, fmt.Errorf("merging scraper spec executor templates: %w", err)
		}
	}

	log.Debug("Scraper job specification", jobSpec)
	decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewBufferString(jobSpec), len(jobSpec))
	if err := decoder.Decode(&job); err != nil {
		return nil, fmt.Errorf("decoding scraper job spec error: %w", err)
	}

	envs := append(executor.RunnerEnvVars, corev1.EnvVar{Name: "RUNNER_CLUSTERID", Value: options.ClusterID})
	if options.ArtifactRequest != nil && options.ArtifactRequest.StorageBucket != "" {
		envs = append(envs, corev1.EnvVar{Name: "RUNNER_BUCKET", Value: options.ArtifactRequest.StorageBucket})
	} else {
		envs = append(envs, corev1.EnvVar{Name: "RUNNER_BUCKET", Value: os.Getenv("STORAGE_BUCKET")})
	}

	if options.HTTPProxy != "" {
		envs = append(envs, corev1.EnvVar{Name: "HTTP_PROXY", Value: options.HTTPProxy})
	}

	if options.HTTPSProxy != "" {
		envs = append(envs, corev1.EnvVar{Name: "HTTPS_PROXY", Value: options.HTTPSProxy})
	}

	for i := range job.Spec.Template.Spec.Containers {
		job.Spec.Template.Spec.Containers[i].Env = append(job.Spec.Template.Spec.Containers[i].Env, envs...)
	}

	return &job, nil
}

// TODO refactor JobOptions to use builder pattern
// TODO extract JobOptions for both container and job executor to common package in separate PR
// NewJobOptions provides job options for templates
func NewJobOptions(log *zap.SugaredLogger, templatesClient templatesv1.Interface, images executor.Images,
	templates executor.Templates, inspector imageinspector.Inspector, serviceAccountNames map[string]string, registry, clusterID, apiURI string,
	execution testkube.Execution, options client.ExecuteOptions, natsUri string, debug bool) (*JobOptions, error) {
	jobOptions := NewJobOptionsFromExecutionOptions(options)
	if execution.PreRunScript != "" || execution.PostRunScript != "" {
		jobOptions.Command = []string{filepath.Join(executor.VolumeDir, EntrypointScriptName)}
		if jobOptions.Image != "" {
			info, err := inspector.Inspect(context.Background(), registry, jobOptions.Image, corev1.PullIfNotPresent, jobOptions.ImagePullSecrets)
			if err == nil {
				if len(execution.Command) == 0 {
					execution.Command = info.Cmd
				}
				execution.ContainerShell = info.Shell
			} else {
				log.Errorw("Docker image inspection error", "error", err)
			}
		}
	}

	jsn, err := json.Marshal(execution)
	if err != nil {
		return nil, err
	}

	jobOptions.Name = execution.Id
	jobOptions.Namespace = execution.TestNamespace
	jobOptions.TestName = execution.TestName
	jobOptions.Jsn = string(jsn)
	jobOptions.InitImage = images.Init
	jobOptions.ScraperImage = images.Scraper

	// options needed for Log sidecar
	if options.Features.LogsV2 {
		// TODO pass them from some config? we dont' have any in this context?
		jobOptions.Debug = debug
		jobOptions.NatsUri = natsUri
		jobOptions.LogSidecarImage = images.LogSidecar
	}

	if jobOptions.JobTemplate == "" {
		jobOptions.JobTemplate = templates.Job
		if jobOptions.JobTemplate == "" {
			jobOptions.JobTemplate = defaultJobTemplate
		}
	}

	if options.ExecutorSpec.JobTemplateReference != "" {
		template, err := templatesClient.Get(options.ExecutorSpec.JobTemplateReference)
		if err != nil {
			return jobOptions, err
		}

		if template.Spec.Type_ != nil && testkube.TemplateType(*template.Spec.Type_) == testkube.CONTAINER_TemplateType {
			jobOptions.JobTemplate = template.Spec.Body
		} else {
			log.Warnw("Not matched template type", "template", options.ExecutorSpec.JobTemplateReference)
		}
	}

	if options.Request.JobTemplateReference != "" {
		template, err := templatesClient.Get(options.Request.JobTemplateReference)
		if err != nil {
			return nil, err
		}

		if template.Spec.Type_ != nil && testkube.TemplateType(*template.Spec.Type_) == testkube.CONTAINER_TemplateType {
			jobOptions.JobTemplate = template.Spec.Body
		} else {
			log.Warnw("Not matched template type", "template", options.Request.JobTemplateReference)
		}
	}

	jobOptions.ScraperTemplate = templates.Scraper
	if options.Request.ScraperTemplateReference != "" {
		template, err := templatesClient.Get(options.Request.ScraperTemplateReference)
		if err != nil {
			return nil, err
		}

		if template.Spec.Type_ != nil && testkube.TemplateType(*template.Spec.Type_) == testkube.SCRAPER_TemplateType {
			jobOptions.ScraperTemplate = template.Spec.Body
		} else {
			log.Warnw("Not matched template type", "template", options.Request.ScraperTemplateReference)
		}
	}

	jobOptions.PvcTemplate = templates.PVC
	if options.Request.PvcTemplateReference != "" {
		template, err := templatesClient.Get(options.Request.PvcTemplateReference)
		if err != nil {
			return nil, err
		}

		if template.Spec.Type_ != nil && testkube.TemplateType(*template.Spec.Type_) == testkube.PVC_TemplateType {
			jobOptions.PvcTemplate = template.Spec.Body
		} else {
			log.Warnw("Not matched template type", "template", options.Request.PvcTemplateReference)
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
	jobOptions.APIURI = apiURI
	return jobOptions, nil
}
