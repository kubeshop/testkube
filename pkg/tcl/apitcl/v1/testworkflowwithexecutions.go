// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package v1

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/repository/result"
	testworkflowmappers "github.com/kubeshop/testkube/pkg/tcl/mapperstcl/testworkflows"
)

func (s *apiTCL) GetTestWorkflowWithExecutionHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		name := c.Params("id")
		errPrefix := fmt.Sprintf("failed to get test workflow '%s' with execution", name)
		if name == "" {
			return s.Error(c, http.StatusBadRequest, errors.New(errPrefix+": id cannot be empty"))
		}
		crWorkflow, err := s.TestWorkflowsClient.Get(name)
		if err != nil {
			return s.ClientError(c, errPrefix, err)
		}

		workflow := testworkflowmappers.MapKubeToAPI(crWorkflow)

		ctx := c.Context()
		execution, err := s.TestWorkflowResults.GetLatestByTestWorkflow(ctx, name)
		if err != nil && !IsNotFound(err) {
			return s.ClientError(c, errPrefix, err)
		}

		return c.JSON(testkube.TestWorkflowWithExecution{
			Workflow:        workflow,
			LatestExecution: execution,
		})
	}
}

func (s *apiTCL) ListTestWorkflowWithExecutionsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		errPrefix := "failed to list test workflows with executions"
		crWorkflows, err := s.getFilteredTestWorkflowList(c)
		if err != nil {
			return s.ClientError(c, errPrefix+": get filtered workflows", err)
		}

		workflows := testworkflowmappers.MapListKubeToAPI(crWorkflows)
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
				iTime = results[i].LatestExecution.StatusAt
			}
			jTime := results[j].Workflow.Created
			if results[j].LatestExecution != nil {
				jTime = results[j].LatestExecution.StatusAt
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
