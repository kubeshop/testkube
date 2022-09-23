package scheduler

import (
	"context"
	"fmt"
	testsv3 "github.com/kubeshop/testkube-operator/apis/tests/v3"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executor/client"
	testsmapper "github.com/kubeshop/testkube/pkg/mapper/tests"
	"github.com/kubeshop/testkube/pkg/secret"
	"github.com/kubeshop/testkube/pkg/workerpool"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
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
		return execution.Errw("can't get current secret uuid: %w", err), nil
	}

	request.TestSecretUUID = secretUUID
	// merge available data into execution options test spec, executor spec, request, test id
	options, err := s.getExecuteOptions(test.Namespace, test.Name, request)
	if err != nil {
		return execution.Errw("can't create valid execution options: %w", err), nil
	}

	// store execution in storage, can be get from API now
	execution = newExecutionFromExecutionOptions(options)
	options.ID = execution.Id

	if err := s.createSecretsReferences(&execution); err != nil {
		return execution.Errw("can't create secret variables `Secret` references: %w", err), nil
	}

	err = s.executionResults.Insert(ctx, execution)
	if err != nil {
		return execution.Errw("can't create new test execution, can't insert into storage: %w", err), nil
	}

	s.logger.Infow("calling executor with options", "options", options.Request)
	execution.Start()

	s.events.Notify(testkube.NewEventStartTest(&execution))

	// update storage with current execution status
	err = s.executionResults.StartExecution(ctx, execution.Id, execution.StartTime)
	if err != nil {
		s.events.Notify(testkube.NewEventEndTestFailed(&execution))
		return execution.Errw("can't execute test, can't insert into storage error: %w", err), nil
	}

	options.HasSecrets = true
	if _, err = s.secretClient.Get(secret.GetMetadataName(execution.TestName)); err != nil {
		if !k8serrors.IsNotFound(err) {
			s.events.Notify(testkube.NewEventEndTestFailed(&execution))
			return execution.Errw("can't get secrets: %w", err), nil
		}

		options.HasSecrets = false
	}

	var result testkube.ExecutionResult

	// sync/async test execution
	if options.Sync {
		result, err = s.executor.ExecuteSync(&execution, options)
	} else {
		result, err = s.executor.Execute(&execution, options)
	}

	// set execution result to one created
	execution.ExecutionResult = &result

	// update storage with current execution status
	if uerr := s.executionResults.UpdateResult(ctx, execution.Id, result); uerr != nil {
		s.events.Notify(testkube.NewEventEndTestFailed(&execution))
		return execution.Errw("update execution error: %w", uerr), nil
	}

	if err != nil {
		s.events.Notify(testkube.NewEventEndTestFailed(&execution))
		return execution.Errw("test execution failed: %w", err), nil
	}

	s.logger.Infow("test started", "executionId", execution.Id, "status", execution.ExecutionResult.Status)

	// notify immediately onlly when sync run otherwise job results handler need notify about test finish
	if options.Sync && execution.ExecutionResult != nil && *execution.ExecutionResult.Status != testkube.RUNNING_ExecutionStatus {
		s.events.Notify(testkube.NewEventEndTestSuccess(&execution))
	}

	return execution, nil
}

func (s *Scheduler) getNextExecutionNumber(testName string) int {
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
		options.Namespace,
		options.TestName,
		options.Request.TestSuiteName,
		options.Request.Name,
		options.TestSpec.Type_,
		options.Request.Number,
		testsmapper.MapTestContentFromSpec(options.TestSpec.Content),
		testkube.NewRunningExecutionResult(),
		options.Request.Variables,
		options.Request.TestSecretUUID,
		options.Request.TestSuiteSecretUUID,
		common.MergeMaps(options.Labels, options.Request.ExecutionLabels),
	)

	execution.Envs = options.Request.Envs
	execution.Args = options.Request.Args
	execution.VariablesFile = options.Request.VariablesFile

	return execution
}

func (s *Scheduler) getExecuteOptions(namespace, id string, request testkube.ExecutionRequest) (options client.ExecuteOptions, err error) {
	// get test content from kubernetes CRs
	testCR, err := s.testsClient.Get(id)
	if err != nil {
		return options, errors.Errorf("can't get test custom resource %v", err)
	}

	test := testsmapper.MapTestCRToAPI(*testCR)

	if test.ExecutionRequest != nil {
		// Test variables lowest priority, then test suite, then test suite execution / test execution
		request.Variables = mergeVariables(test.ExecutionRequest.Variables, request.Variables)
		// Combine test executor args with execution args
		request.Args = append(request.Args, test.ExecutionRequest.Args...)
		request.Envs = mergeEnvs(request.Envs, test.ExecutionRequest.Envs)
		request.SecretEnvs = mergeEnvs(request.SecretEnvs, test.ExecutionRequest.SecretEnvs)
		if request.VariablesFile == "" && test.ExecutionRequest.VariablesFile != "" {
			request.VariablesFile = test.ExecutionRequest.VariablesFile
		}

		if request.HttpProxy == "" && test.ExecutionRequest.HttpProxy != "" {
			request.HttpProxy = test.ExecutionRequest.HttpProxy
		}

		if request.HttpsProxy == "" && test.ExecutionRequest.HttpsProxy != "" {
			request.HttpsProxy = test.ExecutionRequest.HttpsProxy
		}
	}

	// get executor from kubernetes CRs
	executorCR, err := s.executorsClient.GetByType(testCR.Spec.Type_)
	if err != nil {
		return options, errors.Errorf("can't get executor spec: %v", err)
	}

	var usernameSecret, tokenSecret *testkube.SecretRef
	if test.Content != nil && test.Content.Repository != nil {
		usernameSecret = test.Content.Repository.UsernameSecret
		tokenSecret = test.Content.Repository.TokenSecret
	}

	return client.ExecuteOptions{
		TestName:       id,
		Namespace:      namespace,
		TestSpec:       testCR.Spec,
		ExecutorName:   executorCR.ObjectMeta.Name,
		ExecutorSpec:   executorCR.Spec,
		Request:        request,
		Sync:           request.Sync,
		Labels:         testCR.Labels,
		UsernameSecret: usernameSecret,
		TokenSecret:    tokenSecret,
		ImageOverride:  request.Image,
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
