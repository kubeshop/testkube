// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package v1

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/datefilter"
	"github.com/kubeshop/testkube/pkg/tcl/repositorytcl/testworkflow"
	"github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/testworkflowcontroller"
)

func (s *apiTCL) StreamTestWorkflowExecutionNotificationsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := c.Context()
		id := c.Params("executionID")
		errPrefix := fmt.Sprintf("failed to stream test workflow execution notifications '%s'", id)

		// Fetch execution from database
		execution, err := s.TestWorkflowResults.Get(ctx, id)
		if err != nil {
			return s.ClientError(c, errPrefix, err)
		}

		// Check for the logs
		ctrl, err := testworkflowcontroller.New(ctx, s.Clientset, s.Namespace, execution.Id, execution.ScheduledAt)
		if err != nil {
			return s.BadRequest(c, errPrefix, "fetching job", err)
		}

		// Initiate processing event stream
		ctx.SetContentType("text/event-stream")
		ctx.Response.Header.Set("Cache-Control", "no-cache")
		ctx.Response.Header.Set("Connection", "keep-alive")
		ctx.Response.Header.Set("Transfer-Encoding", "chunked")

		// Stream the notifications
		ctx.SetBodyStreamWriter(func(w *bufio.Writer) {
			_ = w.Flush()
			enc := json.NewEncoder(w)

			for n := range ctrl.Watch(ctx).Stream(ctx).Channel() {
				if n.Error == nil {
					_ = enc.Encode(n.Value)
					_, _ = fmt.Fprintf(w, "\n")
					_ = w.Flush()
				}
			}
		})

		return nil
	}
}

func (s *apiTCL) StreamTestWorkflowExecutionNotificationsWebSocketHandler() fiber.Handler {
	return websocket.New(func(c *websocket.Conn) {
		ctx, ctxCancel := context.WithCancel(context.Background())
		id := c.Params("executionID")

		// Stop reading when the WebSocket connection is already closed
		originalClose := c.CloseHandler()
		c.SetCloseHandler(func(code int, text string) error {
			ctxCancel()
			return originalClose(code, text)
		})
		defer c.Conn.Close()

		// Fetch execution from database
		execution, err := s.TestWorkflowResults.Get(ctx, id)
		if err != nil {
			return
		}

		// Check for the logs TODO: Load from the database if possible
		ctrl, err := testworkflowcontroller.New(ctx, s.Clientset, s.Namespace, execution.Id, execution.ScheduledAt)
		if err != nil {
			return
		}

		for n := range ctrl.Watch(ctx).Stream(ctx).Channel() {
			if n.Error == nil {
				_ = c.WriteJSON(n.Value)
			}
		}
	})
}

func (s *apiTCL) ListTestWorkflowExecutionsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		errPrefix := "failed to list test workflow executions"

		filter := getWorkflowExecutionsFilterFromRequest(c)

		executions, err := s.TestWorkflowResults.GetExecutionsSummary(c.Context(), filter)
		if err != nil {
			return s.ClientError(c, errPrefix+": get execution results", err)
		}

		executionTotals, err := s.TestWorkflowResults.GetExecutionsTotals(c.Context(), testworkflow.NewExecutionsFilter().WithName(filter.Name()))
		if err != nil {
			return s.ClientError(c, errPrefix+": get totals", err)
		}

		filterTotals := *filter.(*testworkflow.FilterImpl)
		filterTotals.WithPage(0).WithPageSize(math.MaxInt32)
		filteredTotals, err := s.TestWorkflowResults.GetExecutionsTotals(c.Context(), filterTotals)
		if err != nil {
			return s.ClientError(c, errPrefix+": get filtered totals", err)
		}

		results := testkube.TestWorkflowExecutionsResult{
			Totals:   &executionTotals,
			Filtered: &filteredTotals,
			Results:  executions,
		}
		return c.JSON(results)
	}
}

func (s *apiTCL) GetTestWorkflowMetricsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		workflowName := c.Params("id")

		const DefaultLimit = 0
		limit, err := strconv.Atoi(c.Query("limit", strconv.Itoa(DefaultLimit)))
		if err != nil {
			limit = DefaultLimit
		}

		const DefaultLastDays = 7
		last, err := strconv.Atoi(c.Query("last", strconv.Itoa(DefaultLastDays)))
		if err != nil {
			last = DefaultLastDays
		}

		metrics, err := s.TestWorkflowResults.GetTestWorkflowMetrics(c.Context(), workflowName, limit, last)
		if err != nil {
			return s.ClientError(c, "get metrics for workflow", err)
		}

		return c.JSON(metrics)
	}
}

func (s *apiTCL) GetTestWorkflowExecutionHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := c.Context()
		id := c.Params("id", "")
		executionID := c.Params("executionID")

		var execution testkube.TestWorkflowExecution
		var err error
		if id == "" {
			execution, err = s.TestWorkflowResults.Get(ctx, executionID)
		} else {
			execution, err = s.TestWorkflowResults.GetByNameAndTestWorkflow(ctx, executionID, id)
		}
		if err != nil {
			return s.ClientError(c, "get execution", err)
		}

		return c.JSON(execution)
	}
}

func (s *apiTCL) GetTestWorkflowExecutionLogsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := c.Context()
		id := c.Params("id", "")
		executionID := c.Params("executionID")

		var execution testkube.TestWorkflowExecution
		var err error
		if id == "" {
			execution, err = s.TestWorkflowResults.Get(ctx, executionID)
		} else {
			execution, err = s.TestWorkflowResults.GetByNameAndTestWorkflow(ctx, executionID, id)
		}
		if err != nil {
			return s.ClientError(c, "get execution", err)
		}

		reader, err := s.TestWorkflowOutput.ReadLog(ctx, executionID, execution.Workflow.Name)
		if err != nil {
			return s.InternalError(c, "can't get log", executionID, err)
		}

		c.Context().SetContentType(mediaTypePlainText)
		_, err = io.Copy(c.Response().BodyWriter(), reader)
		return err
	}
}

func (s *apiTCL) AbortTestWorkflowExecutionHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := c.Context()
		name := c.Params("id")
		executionID := c.Params("executionID")
		errPrefix := fmt.Sprintf("failed to abort test workflow execution '%s'", executionID)

		var execution testkube.TestWorkflowExecution
		var err error
		if name == "" {
			execution, err = s.TestWorkflowResults.Get(ctx, executionID)
		} else {
			execution, err = s.TestWorkflowResults.GetByNameAndTestWorkflow(ctx, executionID, name)
		}
		if err != nil {
			return s.ClientError(c, errPrefix, err)
		}

		if execution.Result != nil && execution.Result.IsFinished() {
			return s.BadRequest(c, errPrefix, "checking execution", errors.New("execution already finished"))
		}

		// Obtain the controller
		ctrl, err := testworkflowcontroller.New(ctx, s.Clientset, s.Namespace, execution.Id, execution.ScheduledAt)
		if err != nil {
			return s.BadRequest(c, errPrefix, "fetching job", err)
		}

		// Abort the execution
		err = ctrl.Abort(context.Background())
		if err != nil {
			return s.ClientError(c, "aborting test workflow execution", err)
		}

		c.Status(http.StatusNoContent)

		return nil
	}
}

func (s *apiTCL) AbortAllTestWorkflowExecutionsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := c.Context()
		name := c.Params("id")
		errPrefix := fmt.Sprintf("failed to abort test workflow executions '%s'", name)

		// Fetch executions
		filter := testworkflow.NewExecutionsFilter().WithName(name).WithStatus(string(testkube.RUNNING_TestWorkflowStatus))
		executions, err := s.TestWorkflowResults.GetExecutions(ctx, filter)
		if err != nil {
			if IsNotFound(err) {
				c.Status(http.StatusNoContent)
				return nil
			}
			return s.ClientError(c, errPrefix, err)
		}

		for _, execution := range executions {
			// Obtain the controller
			ctrl, err := testworkflowcontroller.New(ctx, s.Clientset, s.Namespace, execution.Id, execution.ScheduledAt)
			if err != nil {
				return s.BadRequest(c, errPrefix, "fetching job", err)
			}

			// Abort the execution
			err = ctrl.Abort(context.Background())
			if err != nil {
				return s.ClientError(c, errPrefix, err)
			}
		}

		c.Status(http.StatusNoContent)

		return nil
	}
}

func (s *apiTCL) ListTestWorkflowExecutionArtifactsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		executionID := c.Params("executionID")
		errPrefix := fmt.Sprintf("failed to list artifacts for test workflow execution %s", executionID)

		execution, err := s.TestWorkflowResults.Get(c.Context(), executionID)
		if err != nil {
			return s.ClientError(c, errPrefix, err)
		}

		files, err := s.ArtifactsStorage.ListFiles(c.Context(), execution.Id, "", "", execution.Workflow.Name)
		if err != nil {
			return s.InternalError(c, errPrefix, "storage client could not list test workflow files", err)
		}

		return c.JSON(files)
	}
}

func (s *apiTCL) GetTestWorkflowArtifactHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		executionID := c.Params("executionID")
		fileName := c.Params("filename")
		errPrefix := fmt.Sprintf("failed to get artifact %s for workflow execution %s", fileName, executionID)

		// TODO fix this someday :) we don't know 15 mins before release why it's working this way
		// remember about CLI client and Dashboard client too!
		unescaped, err := url.QueryUnescape(fileName)
		if err == nil {
			fileName = unescaped
		}
		unescaped, err = url.QueryUnescape(fileName)
		if err == nil {
			fileName = unescaped
		}
		//// quickfix end

		execution, err := s.TestWorkflowResults.Get(c.Context(), executionID)
		if err != nil {
			return s.ClientError(c, errPrefix, err)
		}

		file, err := s.ArtifactsStorage.DownloadFile(c.Context(), fileName, execution.Id, "", "", execution.Workflow.Name)
		if err != nil {
			return s.InternalError(c, errPrefix, "could not download file", err)
		}

		return c.SendStream(file)
	}
}

func (s *apiTCL) GetTestWorkflowArtifactArchiveHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		executionID := c.Params("executionID")
		query := c.Request().URI().QueryString()
		errPrefix := fmt.Sprintf("failed to get artifact archive for test workflow execution %s", executionID)

		values, err := url.ParseQuery(string(query))
		if err != nil {
			return s.BadRequest(c, errPrefix, "could not parse query string", err)
		}

		execution, err := s.TestWorkflowResults.Get(c.Context(), executionID)
		if err != nil {
			return s.ClientError(c, errPrefix, err)
		}

		archive, err := s.ArtifactsStorage.DownloadArchive(c.Context(), execution.Id, values["mask"])
		if err != nil {
			return s.InternalError(c, errPrefix, "could not download workflow artifact archive", err)
		}

		return c.SendStream(archive)
	}
}

func (s *apiTCL) GetTestWorkflowNotificationsStream(ctx context.Context, executionID string) (chan testkube.TestWorkflowExecutionNotification, error) {
	// Load the execution
	execution, err := s.TestWorkflowResults.Get(ctx, executionID)
	if err != nil {
		return nil, err
	}

	// Check for the logs
	ctrl, err := testworkflowcontroller.New(ctx, s.Clientset, s.Namespace, execution.Id, execution.ScheduledAt)
	if err != nil {
		return nil, err
	}

	// Stream the notifications
	ch := make(chan testkube.TestWorkflowExecutionNotification)
	go func() {
		for n := range ctrl.Watch(ctx).Stream(ctx).Channel() {
			if n.Error == nil {
				ch <- n.Value.ToInternal()
			}
		}
		close(ch)
	}()
	return ch, nil
}

func getWorkflowExecutionsFilterFromRequest(c *fiber.Ctx) testworkflow.Filter {
	filter := testworkflow.NewExecutionsFilter()
	name := c.Params("id", "")
	if name != "" {
		filter = filter.WithName(name)
	}

	textSearch := c.Query("textSearch", "")
	if textSearch != "" {
		filter = filter.WithTextSearch(textSearch)
	}

	page, err := strconv.Atoi(c.Query("page", ""))
	if err == nil {
		filter = filter.WithPage(page)
	}

	pageSize, err := strconv.Atoi(c.Query("pageSize", ""))
	if err == nil && pageSize != 0 {
		filter = filter.WithPageSize(pageSize)
	}

	status := c.Query("status", "")
	if status != "" {
		filter = filter.WithStatus(status)
	}

	last, err := strconv.Atoi(c.Query("last", "0"))
	if err == nil && last != 0 {
		filter = filter.WithLastNDays(last)
	}

	dFilter := datefilter.NewDateFilter(c.Query("startDate", ""), c.Query("endDate", ""))
	if dFilter.IsStartValid {
		filter = filter.WithStartDate(dFilter.Start)
	}

	if dFilter.IsEndValid {
		filter = filter.WithEndDate(dFilter.End)
	}

	selector := c.Query("selector")
	if selector != "" {
		filter = filter.WithSelector(selector)
	}

	return filter
}
