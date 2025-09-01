package v1

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/kubeshop/testkube/internal/app/api/apiutils"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/repository/result"
	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
)

func (s *TestkubeAPI) GetTestWorkflowWithExecutionHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := c.Context()
		environmentId := s.getEnvironmentId()

		name := c.Params("id")
		errPrefix := fmt.Sprintf("failed to get test workflow '%s' with execution", name)
		if name == "" {
			return s.Error(c, http.StatusBadRequest, errors.New(errPrefix+": id cannot be empty"))
		}
		workflow, err := s.TestWorkflowsClient.Get(ctx, environmentId, name)
		if err != nil {
			return s.ClientError(c, errPrefix, err)
		}

		execution, err := s.TestWorkflowResults.GetLatestByTestWorkflow(ctx, name, testworkflow.LatestSortByScheduledAt)
		if err != nil && !apiutils.IsNotFound(err) {
			return s.ClientError(c, errPrefix, err)
		}

		return c.JSON(testkube.TestWorkflowWithExecution{
			Workflow:        workflow,
			LatestExecution: execution,
		})
	}
}

func (s *TestkubeAPI) ListTestWorkflowWithExecutionsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		errPrefix := "failed to list test workflows with executions"
		workflows, err := s.getFilteredTestWorkflowList(c)
		if err != nil {
			return s.ClientError(c, errPrefix+": get filtered workflows", err)
		}

		ctx := c.Context()
		results := make([]testkube.TestWorkflowWithExecutionSummary, 0, len(workflows))
		workflowNames := make([]string, len(workflows))
		for i := range workflows {
			workflowNames[i] = workflows[i].Name
		}

		executions, err := s.TestWorkflowResults.GetLatestByTestWorkflows(ctx, workflowNames)
		if err != nil {
			return s.ClientError(c, errPrefix+": getting latest executions", err)
		}
		executionMap := make(map[string]testkube.TestWorkflowExecutionSummary, len(executions))
		for i := range executions {
			executionMap[executions[i].Workflow.Name] = executions[i]
		}

		for i := range workflows {
			if execution, ok := executionMap[workflows[i].Name]; ok {
				results = append(results, testkube.TestWorkflowWithExecutionSummary{
					Workflow:        &workflows[i],
					LatestExecution: &execution,
				})
			} else {
				results = append(results, testkube.TestWorkflowWithExecutionSummary{
					Workflow: &workflows[i],
				})
			}
		}

		sort.Slice(results, func(i, j int) bool {
			iTime := results[i].Workflow.Created
			if results[i].LatestExecution != nil {
				iTime = results[i].LatestExecution.ScheduledAt
			}
			jTime := results[j].Workflow.Created
			if results[j].LatestExecution != nil {
				jTime = results[j].LatestExecution.ScheduledAt
			}
			return iTime.After(jTime)
		})

		status := c.Query("status")
		if status != "" {
			statusList, err := testkube.ParseTestWorkflowStatusList(status, ",")
			if err != nil {
				return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: execution status filter invalid: %w", errPrefix, err))
			}

			statusMap := statusList.ToMap()
			// filter items array
			for i := len(results) - 1; i >= 0; i-- {
				if results[i].LatestExecution != nil && results[i].LatestExecution.Result.Status != nil {
					if _, ok := statusMap[*results[i].LatestExecution.Result.Status]; ok {
						continue
					}
				}

				results = append(results[:i], results[i+1:]...)
			}
		}

		var page, pageSize int
		pageParam := c.Query("page", "")
		if pageParam != "" {
			pageSize = result.PageDefaultLimit
			page, err = strconv.Atoi(pageParam)
			if err != nil {
				return s.BadRequest(c, errPrefix, "workflow page filter invalid", err)
			}
		}

		pageSizeParam := c.Query("pageSize", "")
		if pageSizeParam != "" {
			pageSize, err = strconv.Atoi(pageSizeParam)
			if err != nil {
				return s.BadRequest(c, errPrefix, "workflow page size filter invalid", err)
			}
		}

		if pageParam != "" || pageSizeParam != "" {
			startPos := page * pageSize
			endPos := (page + 1) * pageSize
			if startPos < len(results) {
				if endPos > len(results) {
					endPos = len(results)
				}

				results = results[startPos:endPos]
			}
		}

		return c.JSON(results)
	}
}
