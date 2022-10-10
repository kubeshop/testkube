package containerexecutor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"

	_ "embed"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executor"
	"github.com/kubeshop/testkube/pkg/executor/client"
	"go.uber.org/zap"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

//go:embed templates/job.tmpl
var defaultJobTemplate string

// NewJobSpec is a method to create new job spec
func NewJobSpec(log *zap.SugaredLogger, options *JobOptions) (*batchv1.Job, error) {
	secretEnvVars := executor.PrepareSecretEnvs(options.SecretEnvs, options.Variables,
		options.UsernameSecret, options.TokenSecret)

	jobTemplate := defaultJobTemplate
	if options.JobTemplate != "" {
		jobTemplate = options.JobTemplate
	}
	tmpl, err := template.New("job").Parse(jobTemplate)
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

func NewJobOptions(initImage, jobTemplate string, execution testkube.Execution, options client.ExecuteOptions) (*JobOptions, error) {
	jsn, err := json.Marshal(execution)
	if err != nil {
		return nil, err
	}

	jobOptions := NewJobOptionsFromExecutionOptions(options)
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
	jobOptions.Envs = options.Request.Envs
	return jobOptions, nil
}
