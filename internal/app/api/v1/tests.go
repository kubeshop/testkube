package v1

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	testsmapper "github.com/kubeshop/testkube/pkg/mapper/tests"
	"github.com/kubeshop/testkube/pkg/rand"

	"k8s.io/apimachinery/pkg/api/errors"
)

// GetTestHandler for getting test object
func (s TestKubeAPI) CreateTestHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var request testkube.ExecutorCreateRequest
		err := c.BodyParser(&request)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		executor := mapExecutorCreateRequestToExecutorCRD(request)
		created, err := s.ExecutorsClient.Create(&executor)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		c.Status(201)
		return c.JSON(created)
	}
}

// GetTestHandler for getting test object
func (s TestKubeAPI) GetTestHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		name := c.Params("id")
		namespace := c.Query("namespace", "testkube")
		crTest, err := s.TestsClient.Get(namespace, name)
		if err != nil {
			if errors.IsNotFound(err) {
				return s.Error(c, http.StatusNotFound, err)
			}

			return s.Error(c, http.StatusBadGateway, err)
		}

		test := testsmapper.MapCRToAPI(*crTest)

		return c.JSON(test)
	}
}

// ListTestsHandler for getting list of all available tests
func (s TestKubeAPI) ListTestsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		s.Log.Debug("Getting scripts list")
		namespace := c.Query("namespace", "testkube")
		crTests, err := s.TestsClient.List(namespace)
		if err != nil {
			return s.Error(c, http.StatusBadGateway, err)
		}

		tests := testsmapper.MapTestListKubeToAPI(*crTests)

		return c.JSON(tests)
	}
}

func (s TestKubeAPI) ExecuteTestHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := context.Background()
		name := c.Params("id")
		namespace := c.Query("namespace", "testkube")

		s.Log.Debugw("getting script ", "name", name)
		crTest, err := s.TestsClient.Get(namespace, name)
		if err != nil {
			if errors.IsNotFound(err) {
				return s.Error(c, http.StatusNotFound, err)
			}

			return s.Error(c, http.StatusBadGateway, err)
		}

		test := testsmapper.MapCRToAPI(*crTest)

		s.Log.Debugw("executing test", "name", name)

		results := s.executeTest(ctx, test)

		c.JSON(results)

		return nil
	}
}

func (s TestKubeAPI) executeTest(ctx context.Context, test testkube.Test) (testExecution testkube.TestExecution) {
	s.Log.Debugw("Got test to execute", "test", test)

	testExecution = testkube.NewStartedTestExecution(rand.Name())
	s.TestExecutionResults.Insert(ctx, testExecution)

	defer func() {
		testExecution.EndTime = time.Now()
		s.TestExecutionResults.EndExecution(ctx, testExecution.Id, time.Now())
	}()

	// compose all steps into one array
	steps := append(test.Before, test.Steps...)
	steps = append(steps, test.After...)

	for _, step := range steps {
		stepResult := s.executeTestStep(ctx, step)
		testExecution.StepResults = append(testExecution.StepResults, stepResult)
		if stepResult.IsFailed() && step.StopOnFailure() {
			return
		}

		s.TestExecutionResults.Update(ctx, testExecution)
	}

	return

}

func (s TestKubeAPI) executeTestStep(ctx context.Context, step testkube.TestStep) (result testkube.TestStepExecutionResult) {
	l := s.Log.With("type", step.Type(), "name", step.FullName())

	switch step.Type() {

	case testkube.EXECUTE_SCRIPT_TestStepType:
		executeScriptStep := step.(testkube.TestStepExecuteScript)
		options, err := s.GetExecuteOptions(executeScriptStep.Namespace, executeScriptStep.Name, testkube.ExecutionRequest{})
		if err != nil {
			return result.Err(err)
		}
		l.Debug("executing script")
		execution := s.executeScript(ctx, options)
		return newTestStepExecutionResult(execution, executeScriptStep)

	case testkube.DELAY_TestStepType:
		l.Debug("delaying execution")
		time.Sleep(time.Millisecond * time.Duration(step.(testkube.TestStepDelay).Duration))

	default:
		result.Err(fmt.Errorf("can't find handler for execution step type: '%v'", step.Type()))
	}

	return
}

func newTestStepExecutionResult(execution testkube.Execution, step testkube.TestStepExecuteScript) (result testkube.TestStepExecutionResult) {
	result.Result = &execution
	result.Script = &testkube.ObjectRef{Name: step.Name, Namespace: step.Namespace}

	return
}
