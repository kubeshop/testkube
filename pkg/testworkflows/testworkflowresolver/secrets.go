package testworkflowresolver

import (
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
)

const (
	GitUsernameKey = "git-username"
	GitTokenKey    = "git-token"
	GitSshKey      = "git-ssh-key"
)

func replacePlainTextCredentialsInContent(content *testworkflowsv1.Content, createSecret func(creds map[string]string) (string, error)) ([]string, error) {
	if content == nil || content.Git == nil {
		return nil, nil
	}

	// Build list of required credentials
	credentials := map[string]string{}
	// TODO: Ensure there is no expression inside
	if content.Git.SshKey != "" {
		credentials[GitSshKey] = content.Git.SshKey
	}
	if content.Git.Username != "" {
		credentials[GitUsernameKey] = content.Git.Username
	}
	if content.Git.Token != "" {
		credentials[GitTokenKey] = content.Git.Token
	}

	if len(credentials) == 0 {
		return nil, nil
	}

	// Attempt creating the credentials secret
	secretName, err := createSecret(credentials)
	if err != nil {
		return nil, errors.Wrap(err, "failed creating secret for Git credentials")
	}
	if secretName == "" {
		return nil, nil
	}

	// Apply the credentials
	if credentials[GitSshKey] != "" {
		content.Git.SshKey = ""
		content.Git.SshKeyFrom = &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
				Key:                  GitSshKey,
			},
		}
	}
	if credentials[GitUsernameKey] != "" {
		content.Git.Username = ""
		content.Git.UsernameFrom = &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
				Key:                  GitUsernameKey,
			},
		}
	}
	if credentials[GitTokenKey] != "" {
		content.Git.Token = ""
		content.Git.TokenFrom = &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
				Key:                  GitTokenKey,
			},
		}
	}

	return []string{secretName}, nil
}

func replacePlainTextCredentialsInService(service *testworkflowsv1.ServiceSpec, createSecret func(creds map[string]string) (string, error)) (secrets []string, err error) {
	if service == nil {
		return
	}
	return replacePlainTextCredentialsInIndependentService(&service.IndependentServiceSpec, createSecret)
}

func replacePlainTextCredentialsInIndependentService(service *testworkflowsv1.IndependentServiceSpec, createSecret func(creds map[string]string) (string, error)) (secrets []string, err error) {
	if service == nil {
		return
	}
	s, err := replacePlainTextCredentialsInContent(service.Content, createSecret)
	return s, errors.Wrap(err, "content")
}

func replacePlainTextCredentialsInStepsList(steps []testworkflowsv1.Step, createSecret func(creds map[string]string) (string, error)) (secrets []string, err error) {
	s := make([]string, 0)
	for i := range steps {
		s, err = replacePlainTextCredentialsInStep(&steps[i], createSecret)
		secrets = append(secrets, s...)
		if err != nil {
			return secrets, errors.Wrapf(err, "%d", i)
		}
	}
	return secrets, nil
}

func replacePlainTextCredentialsInIndependentStepsList(steps []testworkflowsv1.IndependentStep, createSecret func(creds map[string]string) (string, error)) (secrets []string, err error) {
	s := make([]string, 0)
	for i := range steps {
		s, err = replacePlainTextCredentialsInIndependentStep(&steps[i], createSecret)
		secrets = append(secrets, s...)
		if err != nil {
			return secrets, errors.Wrapf(err, "%d", i)
		}
	}
	return secrets, nil
}

func replacePlainTextCredentialsInServicesMap(services map[string]testworkflowsv1.ServiceSpec, createSecret func(creds map[string]string) (string, error)) (secrets []string, err error) {
	s := make([]string, 0)
	for k, svc := range services {
		s, err = replacePlainTextCredentialsInService(&svc, createSecret)
		secrets = append(secrets, s...)
		services[k] = svc
		if err != nil {
			return s, errors.Wrapf(err, k)
		}
	}
	return secrets, nil
}

func replacePlainTextCredentialsInIndependentServicesMap(services map[string]testworkflowsv1.IndependentServiceSpec, createSecret func(creds map[string]string) (string, error)) (secrets []string, err error) {
	s := make([]string, 0)
	for k, svc := range services {
		s, err = replacePlainTextCredentialsInIndependentService(&svc, createSecret)
		secrets = append(secrets, s...)
		services[k] = svc
		if err != nil {
			return s, errors.Wrapf(err, k)
		}
	}
	return secrets, nil
}

func replacePlainTextCredentialsInParallel(parallel *testworkflowsv1.StepParallel, createSecret func(creds map[string]string) (string, error)) (secrets []string, err error) {
	if parallel == nil {
		return
	}
	return replacePlainTextCredentialsInWorkflowSpec(&parallel.TestWorkflowSpec, createSecret)
}

func replacePlainTextCredentialsInIndependentParallel(parallel *testworkflowsv1.IndependentStepParallel, createSecret func(creds map[string]string) (string, error)) (secrets []string, err error) {
	if parallel == nil {
		return
	}
	return replacePlainTextCredentialsInTemplateSpec(&parallel.TestWorkflowTemplateSpec, createSecret)
}

func replacePlainTextCredentialsInStep(step *testworkflowsv1.Step, createSecret func(creds map[string]string) (string, error)) (secrets []string, err error) {
	if step == nil {
		return
	}

	s, err := replacePlainTextCredentialsInContent(step.Content, createSecret)
	secrets = append(secrets, s...)
	if err != nil {
		return s, errors.Wrap(err, "content")
	}

	s, err = replacePlainTextCredentialsInServicesMap(step.Services, createSecret)
	secrets = append(secrets, s...)
	if err != nil {
		return secrets, errors.Wrap(err, "services")
	}

	s, err = replacePlainTextCredentialsInParallel(step.Parallel, createSecret)
	secrets = append(secrets, s...)
	if err != nil {
		return s, errors.Wrap(err, "parallel")
	}

	s, err = replacePlainTextCredentialsInStepsList(step.Setup, createSecret)
	secrets = append(secrets, s...)
	if err != nil {
		return secrets, errors.Wrap(err, "setup")
	}

	s, err = replacePlainTextCredentialsInStepsList(step.Steps, createSecret)
	secrets = append(secrets, s...)
	if err != nil {
		return secrets, errors.Wrap(err, "steps")
	}

	return secrets, nil
}

func replacePlainTextCredentialsInIndependentStep(step *testworkflowsv1.IndependentStep, createSecret func(creds map[string]string) (string, error)) (secrets []string, err error) {
	if step == nil {
		return
	}

	s, err := replacePlainTextCredentialsInContent(step.Content, createSecret)
	secrets = append(secrets, s...)
	if err != nil {
		return s, errors.Wrap(err, "content")
	}

	s, err = replacePlainTextCredentialsInIndependentServicesMap(step.Services, createSecret)
	secrets = append(secrets, s...)
	if err != nil {
		return secrets, errors.Wrap(err, "services")
	}

	s, err = replacePlainTextCredentialsInIndependentParallel(step.Parallel, createSecret)
	secrets = append(secrets, s...)
	if err != nil {
		return s, errors.Wrap(err, "parallel")
	}

	s, err = replacePlainTextCredentialsInIndependentStepsList(step.Setup, createSecret)
	secrets = append(secrets, s...)
	if err != nil {
		return secrets, errors.Wrap(err, "setup")
	}

	s, err = replacePlainTextCredentialsInIndependentStepsList(step.Steps, createSecret)
	secrets = append(secrets, s...)
	if err != nil {
		return secrets, errors.Wrap(err, "steps")
	}

	return secrets, nil
}

func replacePlainTextCredentialsInWorkflowSpec(spec *testworkflowsv1.TestWorkflowSpec, createSecret func(creds map[string]string) (string, error)) (secrets []string, err error) {
	s, err := replacePlainTextCredentialsInContent(spec.Content, createSecret)
	secrets = append(secrets, s...)
	if err != nil {
		return s, errors.Wrap(err, "content")
	}

	s, err = replacePlainTextCredentialsInServicesMap(spec.Services, createSecret)
	secrets = append(secrets, s...)
	if err != nil {
		return secrets, errors.Wrap(err, "services")
	}

	s, err = replacePlainTextCredentialsInStepsList(spec.Setup, createSecret)
	secrets = append(secrets, s...)
	if err != nil {
		return secrets, errors.Wrap(err, "setup")
	}

	s, err = replacePlainTextCredentialsInStepsList(spec.Steps, createSecret)
	secrets = append(secrets, s...)
	if err != nil {
		return secrets, errors.Wrap(err, "steps")
	}

	s, err = replacePlainTextCredentialsInStepsList(spec.After, createSecret)
	secrets = append(secrets, s...)
	if err != nil {
		return secrets, errors.Wrap(err, "after")
	}

	return secrets, nil
}

func replacePlainTextCredentialsInTemplateSpec(spec *testworkflowsv1.TestWorkflowTemplateSpec, createSecret func(creds map[string]string) (string, error)) (secrets []string, err error) {
	s, err := replacePlainTextCredentialsInContent(spec.Content, createSecret)
	secrets = append(secrets, s...)
	if err != nil {
		return s, errors.Wrap(err, "content")
	}

	s, err = replacePlainTextCredentialsInIndependentServicesMap(spec.Services, createSecret)
	secrets = append(secrets, s...)
	if err != nil {
		return secrets, errors.Wrap(err, "services")
	}

	s, err = replacePlainTextCredentialsInIndependentStepsList(spec.Setup, createSecret)
	secrets = append(secrets, s...)
	if err != nil {
		return secrets, errors.Wrap(err, "setup")
	}

	s, err = replacePlainTextCredentialsInIndependentStepsList(spec.Steps, createSecret)
	secrets = append(secrets, s...)
	if err != nil {
		return secrets, errors.Wrap(err, "steps")
	}

	s, err = replacePlainTextCredentialsInIndependentStepsList(spec.After, createSecret)
	secrets = append(secrets, s...)
	if err != nil {
		return secrets, errors.Wrap(err, "after")
	}

	return secrets, nil
}

func ReplacePlainTextCredentialsInWorkflow(workflow *testworkflowsv1.TestWorkflow, createSecret func(creds map[string]string) (string, error)) (secrets []string, err error) {
	if workflow == nil {
		return
	}
	secrets, err = replacePlainTextCredentialsInWorkflowSpec(&workflow.Spec, createSecret)
	return secrets, errors.Wrap(err, "spec")
}

func ReplacePlainTextCredentialsInTemplate(template *testworkflowsv1.TestWorkflowTemplate, createSecret func(creds map[string]string) (string, error)) (secrets []string, err error) {
	if template == nil {
		return
	}
	secrets, err = replacePlainTextCredentialsInTemplateSpec(&template.Spec, createSecret)
	return secrets, errors.Wrap(err, "spec")
}
