package scheduler

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"

	testsv3 "github.com/kubeshop/testkube-operator/apis/tests/v3"
	testsourcev1 "github.com/kubeshop/testkube-operator/apis/testsource/v1"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executor"
	"github.com/kubeshop/testkube/pkg/executor/client"
	testsmapper "github.com/kubeshop/testkube/pkg/mapper/tests"
	"github.com/kubeshop/testkube/pkg/workerpool"
)

const (
	containerType = "container"
)

func (s *Scheduler) PrepareTestRequests(work []testsv3.Test, request testkube.ExecutionRequest) []workerpool.Request[
	testkube.Test, testkube.ExecutionRequest, testkube.Execution] {
	requests := make([]workerpool.Request[testkube.Test, testkube.ExecutionRequest, testkube.Execution], len(work))
	for i := range work {
		requests[i] = workerpool.Request[testkube.Test, testkube.ExecutionRequest, testkube.Execution]{
			Object:  testsmapper.MapTestCRToAPI(work[i]),
			Options: request,
			ExecFn:  s.executeTest,
		}
	}

	return requests
}

func (s *Scheduler) executeTest(ctx context.Context, test testkube.Test, request testkube.ExecutionRequest) (
	execution testkube.Execution, err error) {
	// generate random execution name in case there is no one set
	// like for docker images
	if request.Name == "" && test.ExecutionRequest != nil && test.ExecutionRequest.Name != "" {
		request.Name = test.ExecutionRequest.Name
	}

	if request.Name == "" {
		request.Name = test.Name
	}

	request.Number = s.getNextExecutionNumber(test.Name)
	request.Name = fmt.Sprintf("%s-%d", request.Name, request.Number)

	// test name + test execution name should be unique
	execution, _ = s.executionResults.GetByNameAndTest(ctx, request.Name, test.Name)
	if execution.Name == request.Name {
		return execution.Err(errors.Errorf("test execution with name %s already exists", request.Name)), nil
	}

	secretUUID, err := s.testsClient.GetCurrentSecretUUID(test.Name)
	if err != nil {
		return execution.Errw(request.Id, "can't get current secret uuid: %w", err), nil
	}

	request.TestSecretUUID = secretUUID
	// merge available data into execution options test spec, executor spec, request, test id
	options, err := s.getExecuteOptions(test.Namespace, test.Name, request)
	if err != nil {
		return execution.Errw(request.Id, "can't create valid execution options: %w", err), nil
	}

	// store execution in storage, can be fetched from API now
	execution = newExecutionFromExecutionOptions(options)
	options.ID = execution.Id

	if err := s.createSecretsReferences(&execution); err != nil {
		return execution.Errw(execution.Id, "can't create secret variables `Secret` references: %w", err), nil
	}

	err = s.executionResults.Insert(ctx, execution)
	if err != nil {
		return execution.Errw(execution.Id, "can't create new test execution, can't insert into storage: %w", err), nil
	}

	s.logger.Infow("calling executor with options", "options", options.Request)

	execution.Start()

	s.events.Notify(testkube.NewEventStartTest(&execution))

	// update storage with current execution status
	err = s.executionResults.StartExecution(ctx, execution.Id, execution.StartTime)
	if err != nil {
		s.events.Notify(testkube.NewEventEndTestFailed(&execution))
		return execution.Errw(execution.Id, "can't execute test, can't insert into storage error: %w", err), nil
	}

	// sync/async test execution
	result, err := s.startTestExecution(ctx, options, &execution)

	// set execution result to one created
	execution.ExecutionResult = result

	// update storage with current execution status
	if uerr := s.executionResults.UpdateResult(ctx, execution.Id, execution); uerr != nil {
		s.events.Notify(testkube.NewEventEndTestFailed(&execution))
		return execution.Errw(execution.Id, "update execution error: %w", uerr), nil
	}

	if err != nil {
		s.events.Notify(testkube.NewEventEndTestFailed(&execution))
		return execution.Errw(execution.Id, "test execution failed: %w", err), nil
	}

	s.logger.Infow("test started", "executionId", execution.Id, "status", execution.ExecutionResult.Status)

	return execution, nil
}

func (s *Scheduler) startTestExecution(ctx context.Context, options client.ExecuteOptions, execution *testkube.Execution) (result *testkube.ExecutionResult, err error) {
	executor := s.getExecutor(options.TestName)
	if options.Sync {
		result, err = executor.ExecuteSync(ctx, execution, options)
	} else {
		result, err = executor.Execute(ctx, execution, options)
	}

	return result, err
}

func (s *Scheduler) getExecutor(testName string) client.Executor {
	testCR, err := s.testsClient.Get(testName)
	if err != nil {
		s.logger.Errorw("can't get test", "test", testName, "error", err)
		return s.executor
	}

	executorCR, err := s.executorsClient.GetByType(testCR.Spec.Type_)
	if err != nil {
		s.logger.Errorw("can't get executor", "test", testName, "error", err)
		return s.executor
	}

	switch executorCR.Spec.ExecutorType {
	case containerType:
		return s.containerExecutor
	default:
		return s.executor
	}
}

func (s *Scheduler) getNextExecutionNumber(testName string) int32 {
	number, err := s.executionResults.GetNextExecutionNumber(context.Background(), testName)
	if err != nil {
		s.logger.Errorw("retrieving latest execution", "error", err)
		return number
	}

	return number
}

// createSecretsReferences strips secrets from text and store it inside model as reference to secret
func (s *Scheduler) createSecretsReferences(execution *testkube.Execution) (err error) {
	secrets := map[string]string{}
	secretName := execution.Id + "-vars"

	for k, v := range execution.Variables {
		if v.IsSecret() {
			obfuscated := execution.Variables[k]
			if v.SecretRef != nil {
				obfuscated.SecretRef = &testkube.SecretRef{
					Namespace: execution.TestNamespace,
					Name:      v.SecretRef.Name,
					Key:       v.SecretRef.Key,
				}
			} else {
				obfuscated.Value = ""
				obfuscated.SecretRef = &testkube.SecretRef{
					Namespace: execution.TestNamespace,
					Name:      secretName,
					Key:       v.Name,
				}

				secrets[v.Name] = v.Value
			}

			execution.Variables[k] = obfuscated
		}
	}

	labels := map[string]string{"executionID": execution.Id, "testName": execution.TestName}

	if len(secrets) > 0 {
		return s.secretClient.Create(
			secretName,
			labels,
			secrets,
		)
	}

	return nil
}

func newExecutionFromExecutionOptions(options client.ExecuteOptions) testkube.Execution {
	execution := testkube.NewExecution(
		options.Request.Id,
		options.Namespace,
		options.TestName,
		options.Request.TestSuiteName,
		options.Request.Name,
		options.TestSpec.Type_,
		int(options.Request.Number),
		testsmapper.MapTestContentFromSpec(options.TestSpec.Content),
		*testkube.NewRunningExecutionResult(),
		options.Request.Variables,
		options.Request.TestSecretUUID,
		options.Request.TestSuiteSecretUUID,
		common.MergeMaps(options.Labels, options.Request.ExecutionLabels),
	)

	execution.Envs = options.Request.Envs
	execution.Command = options.Request.Command
	execution.Args = options.Request.Args
	execution.IsVariablesFileUploaded = options.Request.IsVariablesFileUploaded
	execution.VariablesFile = options.Request.VariablesFile
	execution.Uploads = options.Request.Uploads
	execution.BucketName = options.Request.BucketName
	execution.ArtifactRequest = options.Request.ArtifactRequest
	execution.PreRunScript = options.Request.PreRunScript
	execution.PostRunScript = options.Request.PostRunScript
	execution.RunningContext = options.Request.RunningContext

	return execution
}

func (s *Scheduler) getExecuteOptions(namespace, id string, request testkube.ExecutionRequest) (options client.ExecuteOptions, err error) {
	// get test content from kubernetes CRs
	testCR, err := s.testsClient.Get(id)
	if err != nil {
		return options, errors.Errorf("can't get test custom resource %v", err)
	}

	if testCR.Spec.Source != "" {
		testSourceCR, err := s.testSourcesClient.Get(testCR.Spec.Source)
		if err != nil {
			return options, errors.Errorf("cannot get test source custom resource: %v", err)
		}

		testCR.Spec = mergeContents(testCR.Spec, testSourceCR.Spec)

		if testSourceCR.Spec.Type_ == "" && testSourceCR.Spec.Repository.Type_ == "git" {
			testCR.Spec.Content.Type_ = testsv3.TestContentType(testkube.TestContentTypeGit)
		}
	}

	if request.ContentRequest != nil {
		testCR.Spec = adjustContent(testCR.Spec, request.ContentRequest)
	}

	test := testsmapper.MapTestCRToAPI(*testCR)

	if test.ExecutionRequest != nil {
		// Test variables lowest priority, then test suite, then test suite execution / test execution
		request.Variables = mergeVariables(test.ExecutionRequest.Variables, request.Variables)

		request.Envs = mergeEnvs(request.Envs, test.ExecutionRequest.Envs)
		request.SecretEnvs = mergeEnvs(request.SecretEnvs, test.ExecutionRequest.SecretEnvs)
		request.EnvConfigMaps = mergeEnvReferences(request.EnvConfigMaps, test.ExecutionRequest.EnvConfigMaps)
		request.EnvSecrets = mergeEnvReferences(request.EnvSecrets, test.ExecutionRequest.EnvSecrets)

		if request.VariablesFile == "" && test.ExecutionRequest.VariablesFile != "" {
			request.VariablesFile = test.ExecutionRequest.VariablesFile
			request.IsVariablesFileUploaded = test.ExecutionRequest.IsVariablesFileUploaded
		}

		var fields = []struct {
			source      string
			destination *string
		}{
			{
				test.ExecutionRequest.HttpProxy,
				&request.HttpProxy,
			},
			{
				test.ExecutionRequest.HttpsProxy,
				&request.HttpsProxy,
			},
			{
				test.ExecutionRequest.JobTemplate,
				&request.JobTemplate,
			},
			{
				test.ExecutionRequest.PreRunScript,
				&request.PreRunScript,
			},
			{
				test.ExecutionRequest.PostRunScript,
				&request.PostRunScript,
			},
			{
				test.ExecutionRequest.ScraperTemplate,
				&request.ScraperTemplate,
			},
			{
				test.ExecutionRequest.ArgsMode,
				&request.ArgsMode,
			},
		}

		for _, field := range fields {
			if *field.destination == "" && field.source != "" {
				*field.destination = field.source
			}
		}

		// Combine test executor args with execution args
		if len(request.Command) == 0 {
			request.Command = test.ExecutionRequest.Command
		}

		if len(request.Args) == 0 {
			request.Args = test.ExecutionRequest.Args
		}

		if request.ActiveDeadlineSeconds == 0 && test.ExecutionRequest.ActiveDeadlineSeconds != 0 {
			request.ActiveDeadlineSeconds = test.ExecutionRequest.ActiveDeadlineSeconds
		}

		request.ArtifactRequest = mergeArtifacts(request.ArtifactRequest, test.ExecutionRequest.ArtifactRequest)
		if request.ArtifactRequest != nil && request.ArtifactRequest.VolumeMountPath == "" {
			request.ArtifactRequest.VolumeMountPath = filepath.Join(executor.VolumeDir, "artifacts")
		}

		s.logger.Infow("checking for negative test change", "test", test.Name, "negativeTest", request.NegativeTest, "isNegativeTestChangedOnRun", request.IsNegativeTestChangedOnRun)
		if !request.IsNegativeTestChangedOnRun {
			s.logger.Infow("setting negative test from test definition", "test", test.Name, "negativeTest", test.ExecutionRequest.NegativeTest)
			request.NegativeTest = test.ExecutionRequest.NegativeTest
		}
	}

	// get executor from kubernetes CRs
	executorCR, err := s.executorsClient.GetByType(testCR.Spec.Type_)
	if err != nil {
		return options, errors.Errorf("can't get executor spec: %v", err)
	}

	var usernameSecret, tokenSecret *testkube.SecretRef
	var certificateSecret string
	if test.Content != nil && test.Content.Repository != nil {
		usernameSecret = test.Content.Repository.UsernameSecret
		tokenSecret = test.Content.Repository.TokenSecret
		certificateSecret = test.Content.Repository.CertificateSecret
	}

	var imagePullSecrets []string
	switch {
	case len(executorCR.Spec.ImagePullSecrets) != 0:
		imagePullSecrets = mapK8sImagePullSecrets(executorCR.Spec.ImagePullSecrets)

	case testCR.Spec.ExecutionRequest != nil &&
		len(testCR.Spec.ExecutionRequest.ImagePullSecrets) != 0:
		imagePullSecrets = mapK8sImagePullSecrets(testCR.Spec.ExecutionRequest.ImagePullSecrets)

	case len(request.ImagePullSecrets) != 0:
		imagePullSecrets = mapImagePullSecrets(request.ImagePullSecrets)
	}

	configMapVars := make(map[string]testkube.Variable, 0)
	for _, configMap := range request.EnvConfigMaps {
		if configMap.Reference == nil || !configMap.MapToVariables {
			continue
		}

		data, err := s.configMapClient.Get(context.Background(), configMap.Reference.Name)
		if err != nil {
			return options, errors.Errorf("can't get config map: %v", err)
		}

		for key := range data {
			configMapVars[key] = testkube.NewConfigMapVariableReference(key, configMap.Reference.Name, key)
		}
	}

	if len(configMapVars) != 0 {
		request.Variables = mergeVariables(configMapVars, request.Variables)
	}

	secretVars := make(map[string]testkube.Variable, 0)
	for _, secret := range request.EnvSecrets {
		if secret.Reference == nil || !secret.MapToVariables {
			continue
		}

		data, err := s.secretClient.Get(secret.Reference.Name)
		if err != nil {
			return options, errors.Errorf("can't get secret: %v", err)
		}

		for key := range data {
			secretVars[key] = testkube.NewSecretVariableReference(key, secret.Reference.Name, key)
		}
	}

	if len(secretVars) != 0 {
		request.Variables = mergeVariables(secretVars, request.Variables)
	}

	if len(request.Command) == 0 {
		request.Command = executorCR.Spec.Command
	}

	if request.ArgsMode == string(testkube.ArgsModeTypeAppend) || request.ArgsMode == "" {
		request.Args = append(executorCR.Spec.Args, request.Args...)
	}

	return client.ExecuteOptions{
		TestName:             id,
		Namespace:            namespace,
		TestSpec:             testCR.Spec,
		ExecutorName:         executorCR.ObjectMeta.Name,
		ExecutorSpec:         executorCR.Spec,
		Request:              request,
		Sync:                 request.Sync,
		Labels:               testCR.Labels,
		UsernameSecret:       usernameSecret,
		TokenSecret:          tokenSecret,
		CertificateSecret:    certificateSecret,
		ImageOverride:        request.Image,
		ImagePullSecretNames: imagePullSecrets,
	}, nil
}

func mergeVariables(vars1 map[string]testkube.Variable, vars2 map[string]testkube.Variable) map[string]testkube.Variable {
	variables := map[string]testkube.Variable{}
	for k, v := range vars1 {
		variables[k] = v
	}

	for k, v := range vars2 {
		variables[k] = v
	}

	return variables
}

func mergeEnvs(envs1 map[string]string, envs2 map[string]string) map[string]string {
	envs := map[string]string{}
	for k, v := range envs1 {
		envs[k] = v
	}

	for k, v := range envs2 {
		envs[k] = v
	}

	return envs
}

func mergeContents(test testsv3.TestSpec, testSource testsourcev1.TestSourceSpec) testsv3.TestSpec {
	if test.Content == nil {
		test.Content = &testsv3.TestContent{}
	}

	if test.Content.Type_ == "" {
		test.Content.Type_ = testsv3.TestContentType(testSource.Type_)
	}

	if test.Content.Data == "" {
		test.Content.Data = testSource.Data
	}

	if test.Content.Uri == "" {
		test.Content.Uri = testSource.Uri
	}

	if testSource.Repository != nil {
		if test.Content.Repository == nil {
			test.Content.Repository = &testsv3.Repository{}
		}

		if test.Content.Repository.Type_ == "" {
			test.Content.Repository.Type_ = testSource.Repository.Type_
		}

		if test.Content.Repository.Uri == "" {
			test.Content.Repository.Uri = testSource.Repository.Uri
		}

		if test.Content.Repository.Branch == "" {
			test.Content.Repository.Branch = testSource.Repository.Branch
		}

		if test.Content.Repository.Commit == "" {
			test.Content.Repository.Commit = testSource.Repository.Commit
		}

		if test.Content.Repository.Path == "" {
			test.Content.Repository.Path = testSource.Repository.Path
		}

		if test.Content.Repository.UsernameSecret == nil && testSource.Repository.UsernameSecret != nil {
			test.Content.Repository.UsernameSecret = &testsv3.SecretRef{
				Name: testSource.Repository.UsernameSecret.Name,
				Key:  testSource.Repository.UsernameSecret.Key,
			}
		}

		if test.Content.Repository.TokenSecret == nil && testSource.Repository.TokenSecret != nil {
			test.Content.Repository.TokenSecret = &testsv3.SecretRef{
				Name: testSource.Repository.TokenSecret.Name,
				Key:  testSource.Repository.TokenSecret.Key,
			}
		}

		if test.Content.Repository.WorkingDir == "" {
			test.Content.Repository.WorkingDir = testSource.Repository.WorkingDir
		}

		if test.Content.Repository.CertificateSecret == "" {
			test.Content.Repository.CertificateSecret = testSource.Repository.CertificateSecret
		}

		if test.Content.Repository.AuthType == "" {
			test.Content.Repository.AuthType = testsv3.GitAuthType(testSource.Repository.AuthType)
		}
	}

	return test
}

// TODO: generics
func mapImagePullSecrets(secrets []testkube.LocalObjectReference) []string {
	var res []string
	for _, secret := range secrets {
		res = append(res, secret.Name)
	}

	return res
}

func mapK8sImagePullSecrets(secrets []v1.LocalObjectReference) []string {
	var res []string
	for _, secret := range secrets {
		res = append(res, secret.Name)
	}

	return res
}

func mergeArtifacts(artifactBase *testkube.ArtifactRequest, artifactAdjust *testkube.ArtifactRequest) *testkube.ArtifactRequest {
	switch {
	case artifactBase == nil && artifactAdjust == nil:
		return nil
	case artifactBase == nil && artifactAdjust != nil:
		return artifactAdjust
	case artifactBase != nil && artifactAdjust == nil:
		return artifactBase
	default:
		if artifactBase.StorageClassName == "" && artifactAdjust.StorageClassName != "" {
			artifactBase.StorageClassName = artifactAdjust.StorageClassName
		}

		if artifactBase.VolumeMountPath == "" && artifactAdjust.VolumeMountPath != "" {
			artifactBase.VolumeMountPath = artifactAdjust.VolumeMountPath
		}

		artifactBase.Dirs = append(artifactBase.Dirs, artifactAdjust.Dirs...)
	}

	return artifactBase
}

func adjustContent(test testsv3.TestSpec, content *testkube.TestContentRequest) testsv3.TestSpec {
	if test.Content == nil {
		return test
	}

	switch testkube.TestContentType(test.Content.Type_) {
	case testkube.TestContentTypeGitFile, testkube.TestContentTypeGitDir, testkube.TestContentTypeGit:
		if test.Content.Repository == nil {
			return test
		}

		if content.Repository != nil {
			if content.Repository.Branch != "" {
				test.Content.Repository.Branch = content.Repository.Branch
			}

			if content.Repository.Commit != "" {
				test.Content.Repository.Commit = content.Repository.Commit
			}

			if content.Repository.Path != "" {
				test.Content.Repository.Path = content.Repository.Path
			}

			if content.Repository.WorkingDir != "" {
				test.Content.Repository.WorkingDir = content.Repository.WorkingDir
			}
		}
	}

	return test
}

func mergeEnvReferences(envs1 []testkube.EnvReference, envs2 []testkube.EnvReference) []testkube.EnvReference {
	envs := make(map[string]testkube.EnvReference, 0)
	for i := range envs1 {
		if envs1[i].Reference == nil {
			continue
		}

		envs[envs1[i].Reference.Name] = envs1[i]
	}

	for i := range envs2 {
		if envs2[i].Reference == nil {
			continue
		}

		if value, ok := envs[envs2[i].Reference.Name]; !ok {
			envs[envs2[i].Reference.Name] = envs2[i]
		} else {
			if !value.Mount {
				value.Mount = envs2[i].Mount
			}

			if value.MountPath == "" {
				value.MountPath = envs2[i].MountPath
			}

			if !value.MapToVariables {
				value.MapToVariables = envs2[i].MapToVariables
			}

			envs[envs2[i].Reference.Name] = value
		}
	}

	res := make([]testkube.EnvReference, 0)
	for key := range envs {
		res = append(res, envs[key])
	}

	return res
}
