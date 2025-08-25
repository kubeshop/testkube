package testworkflowresolver

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
)

const (
	ComputedKeyword = "<computed>"
	GitUsernameKey  = "git-username"
	GitTokenKey     = "git-token"
	GitSshKey       = "git-ssh-key"
	FileKey         = "file"
	EnvVarKey       = "env"
)

func isComputed(source *corev1.EnvVarSource) bool {
	return source != nil && source.SecretKeyRef != nil && source.SecretKeyRef.Name == ComputedKeyword
}

func extractCredentialsInContent(content *testworkflowsv1.Content, externalize func(key, value string) (*corev1.EnvVarSource, error)) error {
	if content == nil || content.Git == nil {
		return nil
	}

	// Replace credentials in the Git repository
	// TODO: Ensure there is no expression inside
	if isComputed(content.Git.SshKeyFrom) {
		source, err := externalize(GitSshKey, content.Git.SshKeyFrom.SecretKeyRef.Key)
		if err != nil {
			return errors.Wrap(err, "failed creating secret for Git credentials")
		}
		content.Git.SshKeyFrom = source
	}
	if isComputed(content.Git.UsernameFrom) {
		source, err := externalize(GitUsernameKey, content.Git.UsernameFrom.SecretKeyRef.Key)
		if err != nil {
			return errors.Wrap(err, "failed creating secret for Git credentials")
		}
		content.Git.UsernameFrom = source
	}
	if isComputed(content.Git.TokenFrom) {
		source, err := externalize(GitTokenKey, content.Git.TokenFrom.SecretKeyRef.Key)
		if err != nil {
			return errors.Wrap(err, "failed creating secret for Git credentials")
		}
		content.Git.TokenFrom = source
	}

	// TODO: Ensure there is no expression inside
	for i := range content.Files {
		if isComputed(content.Files[i].ContentFrom) {
			source, err := externalize(FileKey, content.Files[i].ContentFrom.SecretKeyRef.Key)
			if err != nil {
				return errors.Wrap(err, "failed creating secret for externalized content file")
			}
			content.Files[i].ContentFrom = source
		}
	}

	return nil
}

var sanitizeNameRe = regexp.MustCompile(`[^a-zA-Z0-9]+`)

func sanitizeName(name string) string {
	return strings.Trim(strings.ToLower(sanitizeNameRe.ReplaceAllString(name, "-")), "-")
}

func extractCredentialsInContainerConfig(container *testworkflowsv1.ContainerConfig, externalize func(key, value string) (*corev1.EnvVarSource, error)) error {
	if container == nil {
		return nil
	}
	for i := range container.Env {
		if isComputed(container.Env[i].ValueFrom) {
			source, err := externalize(fmt.Sprintf("%s-%s", EnvVarKey, sanitizeName(container.Env[i].Name)), container.Env[i].ValueFrom.SecretKeyRef.Key)
			if err != nil {
				return errors.Wrap(err, "failed creating secret for externalized environment variable")
			}
			container.Env[i].ValueFrom = source
		}
	}
	return nil
}

func extractCredentialsInService(service *testworkflowsv1.ServiceSpec, externalize func(key, value string) (*corev1.EnvVarSource, error)) (err error) {
	if service == nil {
		return
	}
	return extractCredentialsInIndependentService(&service.IndependentServiceSpec, externalize)
}

func extractCredentialsInStepRun(step *testworkflowsv1.StepRun, externalize func(key, value string) (*corev1.EnvVarSource, error)) (err error) {
	if step == nil {
		return
	}
	err = extractCredentialsInContainerConfig(&step.ContainerConfig, externalize)
	return errors.Wrap(err, "run")
}

func extractCredentialsInIndependentService(service *testworkflowsv1.IndependentServiceSpec, externalize func(key, value string) (*corev1.EnvVarSource, error)) (err error) {
	if service == nil {
		return
	}
	err = extractCredentialsInContainerConfig(&service.ContainerConfig, externalize)
	if err != nil {
		return err
	}
	err = extractCredentialsInStepRun(&service.StepRun, externalize)
	if err != nil {
		return err
	}
	err = extractCredentialsInContent(service.Content, externalize)
	return errors.Wrap(err, "content")
}

func extractCredentialsInStepsList(steps []testworkflowsv1.Step, externalize func(key, value string) (*corev1.EnvVarSource, error)) (err error) {
	for i := range steps {
		err = extractCredentialsInStep(&steps[i], externalize)
		if err != nil {
			return errors.Wrapf(err, "%d", i)
		}
	}
	return nil
}

func extractCredentialsInIndependentStepsList(steps []testworkflowsv1.IndependentStep, externalize func(key, value string) (*corev1.EnvVarSource, error)) (err error) {
	for i := range steps {
		err = extractCredentialsInIndependentStep(&steps[i], externalize)
		if err != nil {
			return errors.Wrapf(err, "%d", i)
		}
	}
	return nil
}

func extractCredentialsInServicesMap(services map[string]testworkflowsv1.ServiceSpec, externalize func(key, value string) (*corev1.EnvVarSource, error)) (err error) {
	for k, svc := range services {
		err = extractCredentialsInService(&svc, externalize)
		services[k] = svc
		if err != nil {
			return errors.Wrapf(err, k)
		}
	}
	return nil
}

func extractCredentialsInIndependentServicesMap(services map[string]testworkflowsv1.IndependentServiceSpec, externalize func(key, value string) (*corev1.EnvVarSource, error)) (err error) {
	for k, svc := range services {
		err = extractCredentialsInIndependentService(&svc, externalize)
		services[k] = svc
		if err != nil {
			return errors.Wrapf(err, k)
		}
	}
	return nil
}

func extractCredentialsInParallel(parallel *testworkflowsv1.StepParallel, externalize func(key, value string) (*corev1.EnvVarSource, error)) (err error) {
	if parallel == nil {
		return
	}
	return extractCredentialsInWorkflowSpec(parallel.NewTestWorkflowSpec(), externalize)
}

func extractCredentialsInIndependentParallel(parallel *testworkflowsv1.IndependentStepParallel, externalize func(key, value string) (*corev1.EnvVarSource, error)) (err error) {
	if parallel == nil {
		return
	}
	return extractCredentialsInTemplateSpec(&parallel.TestWorkflowTemplateSpec, externalize)
}

func extractCredentialsInStep(step *testworkflowsv1.Step, externalize func(key, value string) (*corev1.EnvVarSource, error)) (err error) {
	if step == nil {
		return
	}

	err = extractCredentialsInContent(step.Content, externalize)
	if err != nil {
		return errors.Wrap(err, "content")
	}

	err = extractCredentialsInStepRun(step.Run, externalize)
	if err != nil {
		return errors.Wrap(err, "run")
	}

	err = extractCredentialsInContainerConfig(step.Container, externalize)
	if err != nil {
		return errors.Wrap(err, "container")
	}

	err = extractCredentialsInServicesMap(step.Services, externalize)
	if err != nil {
		return errors.Wrap(err, "services")
	}

	err = extractCredentialsInParallel(step.Parallel, externalize)
	if err != nil {
		return errors.Wrap(err, "parallel")
	}

	err = extractCredentialsInStepsList(step.Setup, externalize)
	if err != nil {
		return errors.Wrap(err, "setup")
	}

	err = extractCredentialsInStepsList(step.Steps, externalize)
	if err != nil {
		return errors.Wrap(err, "steps")
	}

	return nil
}

func extractCredentialsInIndependentStep(step *testworkflowsv1.IndependentStep, externalize func(key, value string) (*corev1.EnvVarSource, error)) (err error) {
	if step == nil {
		return
	}

	err = extractCredentialsInContent(step.Content, externalize)
	if err != nil {
		return errors.Wrap(err, "content")
	}

	err = extractCredentialsInStepRun(step.Run, externalize)
	if err != nil {
		return errors.Wrap(err, "run")
	}

	err = extractCredentialsInContainerConfig(step.Container, externalize)
	if err != nil {
		return errors.Wrap(err, "container")
	}

	err = extractCredentialsInIndependentServicesMap(step.Services, externalize)
	if err != nil {
		return errors.Wrap(err, "services")
	}

	err = extractCredentialsInIndependentParallel(step.Parallel, externalize)
	if err != nil {
		return errors.Wrap(err, "parallel")
	}

	err = extractCredentialsInIndependentStepsList(step.Setup, externalize)
	if err != nil {
		return errors.Wrap(err, "setup")
	}

	err = extractCredentialsInIndependentStepsList(step.Steps, externalize)
	if err != nil {
		return errors.Wrap(err, "steps")
	}

	return nil
}

func extractCredentialsInWorkflowSpec(spec *testworkflowsv1.TestWorkflowSpec, externalize func(key, value string) (*corev1.EnvVarSource, error)) (err error) {
	err = extractCredentialsInContent(spec.Content, externalize)
	if err != nil {
		return errors.Wrap(err, "content")
	}

	err = extractCredentialsInContainerConfig(spec.Container, externalize)
	if err != nil {
		return errors.Wrap(err, "container")
	}

	err = extractCredentialsInServicesMap(spec.Services, externalize)
	if err != nil {
		return errors.Wrap(err, "services")
	}

	err = extractCredentialsInStepsList(spec.Setup, externalize)
	if err != nil {
		return errors.Wrap(err, "setup")
	}

	err = extractCredentialsInStepsList(spec.Steps, externalize)
	if err != nil {
		return errors.Wrap(err, "steps")
	}

	err = extractCredentialsInStepsList(spec.After, externalize)
	if err != nil {
		return errors.Wrap(err, "after")
	}

	return nil
}

func extractCredentialsInTemplateSpec(spec *testworkflowsv1.TestWorkflowTemplateSpec, externalize func(key, value string) (*corev1.EnvVarSource, error)) (err error) {
	err = extractCredentialsInContent(spec.Content, externalize)
	if err != nil {
		return errors.Wrap(err, "content")
	}

	err = extractCredentialsInContainerConfig(spec.Container, externalize)
	if err != nil {
		return errors.Wrap(err, "container")
	}

	err = extractCredentialsInIndependentServicesMap(spec.Services, externalize)
	if err != nil {
		return errors.Wrap(err, "services")
	}

	err = extractCredentialsInIndependentStepsList(spec.Setup, externalize)
	if err != nil {
		return errors.Wrap(err, "setup")
	}

	err = extractCredentialsInIndependentStepsList(spec.Steps, externalize)
	if err != nil {
		return errors.Wrap(err, "steps")
	}

	err = extractCredentialsInIndependentStepsList(spec.After, externalize)
	if err != nil {
		return errors.Wrap(err, "after")
	}

	return nil
}

func ExtractCredentialsInWorkflow(workflow *testworkflowsv1.TestWorkflow, externalize func(key, value string) (*corev1.EnvVarSource, error)) (err error) {
	if workflow == nil {
		return
	}
	err = extractCredentialsInWorkflowSpec(&workflow.Spec, externalize)
	return errors.Wrap(err, "spec")
}

func ExtractCredentialsInTemplate(template *testworkflowsv1.TestWorkflowTemplate, externalize func(key, value string) (*corev1.EnvVarSource, error)) (err error) {
	if template == nil {
		return
	}
	err = extractCredentialsInTemplateSpec(&template.Spec, externalize)
	return errors.Wrap(err, "spec")
}
