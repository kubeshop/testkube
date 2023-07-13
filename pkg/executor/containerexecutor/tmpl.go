package containerexecutor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"text/template"

	_ "embed"

	"go.uber.org/zap"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
	"sigs.k8s.io/kustomize/kyaml/yaml/merge2"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executor"
	"github.com/kubeshop/testkube/pkg/executor/client"
	"github.com/kubeshop/testkube/pkg/executor/env"
	"github.com/kubeshop/testkube/pkg/secret"
	"github.com/kubeshop/testkube/pkg/skopeo"
)

//go:embed templates/job.tmpl
var defaultJobTemplate string

// NewExecutorJobSpec is a method to create new executor job spec
func NewExecutorJobSpec(log *zap.SugaredLogger, options *JobOptions) (*batchv1.Job, error) {
	envManager := env.NewManager()
	secretEnvVars := append(envManager.PrepareSecrets(options.SecretEnvs, options.Variables),
		envManager.PrepareGitCredentials(options.UsernameSecret, options.TokenSecret)...)

	tmpl, err := template.New("job").Parse(options.JobTemplate)
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
		tmplExt, err := template.New("jobExt").Parse(options.JobTemplateExtensions)
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

	log.Debug("Executor job specification", jobSpec)
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
	envs = append(envs, secretEnvVars...)
	if options.HTTPProxy != "" {
		envs = append(envs, corev1.EnvVar{Name: "HTTP_PROXY", Value: options.HTTPProxy})
	}

	if options.HTTPSProxy != "" {
		envs = append(envs, corev1.EnvVar{Name: "HTTPS_PROXY", Value: options.HTTPSProxy})
	}

	envs = append(envs, envManager.PrepareEnvs(options.Envs, options.Variables)...)

	for i := range job.Spec.Template.Spec.InitContainers {
		job.Spec.Template.Spec.InitContainers[i].Env = append(job.Spec.Template.Spec.InitContainers[i].Env, envs...)
	}

	for i := range job.Spec.Template.Spec.Containers {
		job.Spec.Template.Spec.Containers[i].Env = append(job.Spec.Template.Spec.Containers[i].Env, envs...)
		// override container image if provided
		if options.ImageOverride != "" {
			job.Spec.Template.Spec.Containers[i].Image = options.ImageOverride
		}
	}

	return &job, nil
}

// NewScraperJobSpec is a method to create new scraper job spec
func NewScraperJobSpec(log *zap.SugaredLogger, options *JobOptions) (*batchv1.Job, error) {
	tmpl, err := template.New("job").Parse(options.ScraperTemplate)
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
		tmplExt, err := template.New("jobExt").Parse(options.ScraperTemplateExtensions)
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

// NewPersistentVolumeClaimSpec is a method to create new persistent volume claim spec
func NewPersistentVolumeClaimSpec(log *zap.SugaredLogger, options *JobOptions) (*corev1.PersistentVolumeClaim, error) {
	tmpl, err := template.New("volume-claim").Parse(options.PVCTemplate)
	if err != nil {
		return nil, fmt.Errorf("creating volume claim spec from pvc template error: %w", err)
	}

	var buffer bytes.Buffer
	if err = tmpl.ExecuteTemplate(&buffer, "volume-claim", options); err != nil {
		return nil, fmt.Errorf("executing volume claim spec pvc template: %w", err)
	}

	var pvc corev1.PersistentVolumeClaim
	pvcSpec := buffer.String()

	log.Debug("Volume claim specification", pvcSpec)
	decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewBufferString(pvcSpec), len(pvcSpec))
	if err := decoder.Decode(&pvc); err != nil {
		return nil, fmt.Errorf("decoding pvc spec error: %w", err)
	}

	return &pvc, nil
}

// InspectDockerImage inspects docker image
func InspectDockerImage(namespace, image string, imageSecrets []string) ([]string, string, error) {
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

	}

	dockerImage, err := inspector.Inspect(image)
	if err != nil {
		return nil, "", err
	}

	return append(dockerImage.Config.Entrypoint, dockerImage.Config.Cmd...), dockerImage.Shell, nil
}

// NewJobOptions provides job options for templates
func NewJobOptions(log *zap.SugaredLogger, images executor.Images, templates executor.Templates,
	serviceAccountName, registry, clusterID string, execution testkube.Execution, options client.ExecuteOptions) (*JobOptions, error) {
	jobOptions := NewJobOptionsFromExecutionOptions(options)
	if execution.PreRunScript != "" || execution.PostRunScript != "" {
		jobOptions.Command = []string{filepath.Join(executor.VolumeDir, "entrypoint.sh")}
		if jobOptions.Image != "" {
			cmd, shell, err := InspectDockerImage(jobOptions.Namespace, jobOptions.Image, jobOptions.ImagePullSecrets)
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
	if jobOptions.JobTemplate == "" {
		jobOptions.JobTemplate = templates.Job
		if jobOptions.JobTemplate == "" {
			jobOptions.JobTemplate = defaultJobTemplate
		}
	}

	jobOptions.ScraperTemplate = templates.Scraper
	jobOptions.PVCTemplate = templates.PVC
	jobOptions.Variables = execution.Variables
	jobOptions.ServiceAccountName = serviceAccountName
	jobOptions.Registry = registry
	jobOptions.ClusterID = clusterID
	return jobOptions, nil
}
