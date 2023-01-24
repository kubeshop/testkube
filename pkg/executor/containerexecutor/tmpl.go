package containerexecutor

import (
	"bytes"
	"encoding/json"
	"fmt"
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
)

//go:embed templates/job.tmpl
var defaultJobTemplate string

// NewExecutorJobSpec is a method to create new executor job spec
func NewExecutorJobSpec(log *zap.SugaredLogger, options *JobOptions) (*batchv1.Job, error) {
	secretEnvVars := executor.PrepareSecretEnvs(options.SecretEnvs, options.Variables,
		options.UsernameSecret, options.TokenSecret)

	tmpl, err := template.New("job").Parse(options.JobTemplate)
	if err != nil {
		return nil, fmt.Errorf("creating job spec from executor template error: %w", err)
	}
	options.Jsn = strings.ReplaceAll(options.Jsn, "'", "''")
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

	env := append(executor.RunnerEnvVars, secretEnvVars...)
	if options.HTTPProxy != "" {
		env = append(env, corev1.EnvVar{Name: "HTTP_PROXY", Value: options.HTTPProxy})
	}

	if options.HTTPSProxy != "" {
		env = append(env, corev1.EnvVar{Name: "HTTPS_PROXY", Value: options.HTTPSProxy})
	}

	for _, variable := range options.Variables {
		if variable.Type_ != nil && *variable.Type_ == testkube.BASIC_VariableType {
			env = append(env, corev1.EnvVar{Name: strings.ToUpper(variable.Name), Value: variable.Value})
		}
	}
	env = append(env, executor.PrepareEnvs(options.Envs)...)

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

	env := executor.RunnerEnvVars
	if options.HTTPProxy != "" {
		env = append(env, corev1.EnvVar{Name: "HTTP_PROXY", Value: options.HTTPProxy})
	}

	if options.HTTPSProxy != "" {
		env = append(env, corev1.EnvVar{Name: "HTTPS_PROXY", Value: options.HTTPSProxy})
	}

	for i := range job.Spec.Template.Spec.Containers {
		job.Spec.Template.Spec.Containers[i].Env = append(job.Spec.Template.Spec.Containers[i].Env, env...)
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

// NewJobOptions provides job options for templates
func NewJobOptions(images executor.Images, templates executor.Templates, serviceAccountName string, execution testkube.Execution, options client.ExecuteOptions) (*JobOptions, error) {
	jsn, err := json.Marshal(execution)
	if err != nil {
		return nil, err
	}

	jobOptions := NewJobOptionsFromExecutionOptions(options)
	jobOptions.Name = execution.Id
	jobOptions.Namespace = execution.TestNamespace
	jobOptions.TestName = execution.TestName
	jobOptions.Jsn = string(jsn)
	jobOptions.InitImage = images.Init
	jobOptions.ScraperImage = images.Scraper
	jobOptions.JobTemplate = templates.Job
	if jobOptions.JobTemplate == "" {
		jobOptions.JobTemplate = defaultJobTemplate
	}

	jobOptions.ScraperTemplate = templates.Scraper
	jobOptions.PVCTemplate = templates.PVC
	jobOptions.Variables = execution.Variables
	jobOptions.ImagePullSecrets = options.ImagePullSecretNames
	jobOptions.Envs = options.Request.Envs
	jobOptions.ServiceAccountName = serviceAccountName
	return jobOptions, nil
}
