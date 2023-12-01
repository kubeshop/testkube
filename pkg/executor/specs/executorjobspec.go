package specs

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"strings"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executor"
	"github.com/kubeshop/testkube/pkg/executor/env"
	"github.com/kubeshop/testkube/pkg/executor/options"
	"github.com/kubeshop/testkube/pkg/utils"
	"go.uber.org/zap"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/kustomize/kyaml/yaml/merge2"

	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

// job executor

// NewExecutorJobSpec is a method to create new executor job spec
func NewExecutorJobSpec(log *zap.SugaredLogger, options options.JobOptions) (*batchv1.Job, error) {
	envManager := env.NewManager()
	secretEnvVars := append(envManager.PrepareSecrets(options.SecretEnvs, options.Variables),
		envManager.PrepareGitCredentials(options.UsernameSecret, options.TokenSecret)...)

	tmpl, err := utils.NewTemplate("job").
		Funcs(template.FuncMap{"vartypeptrtostring": testkube.VariableTypeString}).
		Parse(options.JobTemplate)
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
		tmplExt, err := utils.NewTemplate("jobExt").
			Funcs(template.FuncMap{"vartypeptrtostring": testkube.VariableTypeString}).
			Parse(options.JobTemplateExtensions)
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
