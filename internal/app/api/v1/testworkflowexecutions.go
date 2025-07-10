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
	"github.com/valyala/fasthttp"

	"github.com/kubeshop/testkube/internal/app/api/apiutils"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/datefilter"
	testworkflow2 "github.com/kubeshop/testkube/pkg/repository/testworkflow"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
)

func (s *TestkubeAPI) streamNotifications(ctx *fasthttp.RequestCtx, id string, notifications executionworkertypes.NotificationsWatcher) {
	// Initiate processing event stream
	ctx.SetContentType("text/event-stream")
	ctx.Response.Header.Set("Cache-Control", "no-cache")
	ctx.Response.Header.Set("Connection", "keep-alive")
	ctx.Response.Header.Set("Transfer-Encoding", "chunked")

	// Stream the notifications
	ctx.SetBodyStreamWriter(func(w *bufio.Writer) {
		err := w.Flush()
		if err != nil {
			s.Log.Errorw("could not flush stream body", "error", err, "id", id)
		}

		enc := json.NewEncoder(w)

		for n := range notifications.Channel() {
			err := enc.Encode(n)
			if err != nil {
				s.Log.Errorw("could not encode value", "error", err, "id", id)
			}

			_, err = fmt.Fprintf(w, "\n")
			if err != nil {
				s.Log.Errorw("could not print new line", "error", err, "id", id)
			}

			err = w.Flush()
			if err != nil {
				s.Log.Errorw("could not flush stream body", "error", err, "id", id)
			}
		}
	})
}

func (s *TestkubeAPI) StreamTestWorkflowExecutionNotificationsHandler() fiber.Handler {
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
		notifications := s.ExecutionWorkerClient.Notifications(ctx, execution.Id, executionworkertypes.NotificationsOptions{
			Hints: executionworkertypes.Hints{
				Namespace:   execution.Namespace,
				ScheduledAt: common.Ptr(execution.ScheduledAt),
				Signature:   execution.Signature,
			},
		})
		if notifications.Err() != nil {
			return s.BadRequest(c, errPrefix, "fetching notifications", notifications.Err())
		}

		s.streamNotifications(ctx, id, notifications)
		return nil
	}
}

func (s *TestkubeAPI) StreamTestWorkflowExecutionServiceNotificationsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := c.Context()
		executionID := c.Params("executionID")
		serviceName := c.Params("serviceName")
		serviceIndex := c.Params("serviceIndex")
		errPrefix := fmt.Sprintf("failed to stream test workflow execution service '%s' instance '%s' notifications '%s'",
			serviceName, serviceIndex, executionID)

		// Fetch execution from database
		execution, err := s.TestWorkflowResults.Get(ctx, executionID)
		if err != nil {
			return s.ClientError(c, errPrefix, err)
		}

		found := false
		if execution.Workflow != nil {
			found = execution.Workflow.HasService(serviceName)
		}

		if !found {
			return s.ClientError(c, errPrefix, errors.New("unknown service for test workflow execution"))
		}

		// Check for the logs
		id := fmt.Sprintf("%s-%s-%s", execution.Id, serviceName, serviceIndex)
		notifications := s.ExecutionWorkerClient.Notifications(ctx, id, executionworkertypes.NotificationsOptions{
			Hints: executionworkertypes.Hints{
				Namespace:   execution.Namespace,
				ScheduledAt: common.Ptr(execution.ScheduledAt),
			},
		})
		if notifications.Err() != nil {
			return s.BadRequest(c, errPrefix, "fetching notifications", notifications.Err())
		}

		s.streamNotifications(ctx, id, notifications)
		return nil
	}
}

func (s *TestkubeAPI) StreamTestWorkflowExecutionParallelStepNotificationsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := c.Context()
		executionID := c.Params("executionID")
		ref := c.Params("ref")
		workerIndex := c.Params("workerIndex")
		errPrefix := fmt.Sprintf("failed to stream test workflow execution parallel step '%s' instance '%s' notifications '%s'",
			ref, workerIndex, executionID)

		// Fetch execution from database
		execution, err := s.TestWorkflowResults.Get(ctx, executionID)
		if err != nil {
			return s.ClientError(c, errPrefix, err)
		}

		reference := execution.GetParallelStepReference(ref)
		if reference == "" {
			return s.ClientError(c, errPrefix, errors.New("unknown parallel step for test workflow execution"))
		}

		// Check for the logs
		id := fmt.Sprintf("%s-%s-%s", execution.Id, reference, workerIndex)
		notifications := s.ExecutionWorkerClient.Notifications(ctx, id, executionworkertypes.NotificationsOptions{
			Hints: executionworkertypes.Hints{
				Namespace:   execution.Namespace,
				ScheduledAt: common.Ptr(execution.ScheduledAt),
			},
		})
		if notifications.Err() != nil {
			return s.BadRequest(c, errPrefix, "fetching notifications", notifications.Err())
		}

		s.streamNotifications(ctx, id, notifications)
		return nil
	}
}

func (s *TestkubeAPI) StreamTestWorkflowExecutionNotificationsWebSocketHandler() fiber.Handler {
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

		// Check for the logs
		notifications := s.ExecutionWorkerClient.Notifications(ctx, execution.Id, executionworkertypes.NotificationsOptions{
			Hints: executionworkertypes.Hints{
				Namespace:   execution.Namespace,
				Signature:   execution.Signature,
				ScheduledAt: common.Ptr(execution.ScheduledAt),
			},
		})
		if notifications.Err() != nil {
			return
		}

		for n := range notifications.Channel() {
			_ = c.WriteJSON(n)
		}
	})
}

func (s *TestkubeAPI) StreamTestWorkflowExecutionServiceNotificationsWebSocketHandler() fiber.Handler {
	return websocket.New(func(c *websocket.Conn) {
		ctx, ctxCancel := context.WithCancel(context.Background())
		executionID := c.Params("executionID")
		serviceName := c.Params("serviceName")
		serviceIndex := c.Params("serviceIndex")

		// Stop reading when the WebSocket connection is already closed
		originalClose := c.CloseHandler()
		c.SetCloseHandler(func(code int, text string) error {
			ctxCancel()
			return originalClose(code, text)
		})
		defer c.Conn.Close()

		// Fetch execution from database
		execution, err := s.TestWorkflowResults.Get(ctx, executionID)
		if err != nil {
			return
		}

		found := false
		if execution.Workflow != nil && execution.Workflow.Spec != nil {
			found = execution.Workflow.HasService(serviceName)
		}

		if !found {
			return
		}

		// Check for the logs
		id := fmt.Sprintf("%s-%s-%s", execution.Id, serviceName, serviceIndex)
		notifications := s.ExecutionWorkerClient.Notifications(ctx, id, executionworkertypes.NotificationsOptions{
			Hints: executionworkertypes.Hints{
				Namespace:   execution.Namespace,
				ScheduledAt: common.Ptr(execution.ScheduledAt),
			},
		})
		if notifications.Err() != nil {
			return
		}

		for n := range notifications.Channel() {
			_ = c.WriteJSON(n)
		}
	})
}

func (s *TestkubeAPI) StreamTestWorkflowExecutionParallelStepNotificationsWebSocketHandler() fiber.Handler {
	return websocket.New(func(c *websocket.Conn) {
		ctx, ctxCancel := context.WithCancel(context.Background())
		executionID := c.Params("executionID")
		ref := c.Params("ref")
		workerIndex := c.Params("workerIndex")

		// Stop reading when the WebSocket connection is already closed
		originalClose := c.CloseHandler()
		c.SetCloseHandler(func(code int, text string) error {
			ctxCancel()
			return originalClose(code, text)
		})
		defer c.Conn.Close()

		// Fetch execution from database
		execution, err := s.TestWorkflowResults.Get(ctx, executionID)
		if err != nil {
			return
		}

		reference := execution.GetParallelStepReference(ref)
		if reference == "" {
			return
		}

		// Check for the logs
		id := fmt.Sprintf("%s-%s-%s", execution.Id, reference, workerIndex)
		// NOTE: usage of notications
		notifications := s.ExecutionWorkerClient.Notifications(ctx, id, executionworkertypes.NotificationsOptions{
			Hints: executionworkertypes.Hints{
				Namespace:   execution.Namespace,
				ScheduledAt: common.Ptr(execution.ScheduledAt),
			},
		})
		if notifications.Err() != nil {
			return
		}

		for n := range notifications.Channel() {
			_ = c.WriteJSON(n)
		}
	})
}

func (s *TestkubeAPI) ListTestWorkflowExecutionsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		errPrefix := "failed to list test workflow executions"

		filter := getWorkflowExecutionsFilterFromRequest(c)

		executions, err := s.TestWorkflowResults.GetExecutionsSummary(c.Context(), filter)
		if err != nil {
			return s.ClientError(c, errPrefix+": get execution results", err)
		}

		executionTotals, err := s.TestWorkflowResults.GetExecutionsTotals(c.Context(), testworkflow2.NewExecutionsFilter().WithName(filter.Name()))
		if err != nil {
			return s.ClientError(c, errPrefix+": get totals", err)
		}

		filterTotals := *filter.(*testworkflow2.FilterImpl)
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

func (s *TestkubeAPI) GetTestWorkflowMetricsHandler() fiber.Handler {
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

func (s *TestkubeAPI) GetTestWorkflowExecutionHandler() fiber.Handler {
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

func (s *TestkubeAPI) GetTestWorkflowExecutionLogsHandler() fiber.Handler {
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

		rc, err := s.TestWorkflowOutput.ReadLog(ctx, executionID, execution.Workflow.Name)
		if err != nil {
			return s.InternalError(c, "can't get log", executionID, err)
		}
		defer rc.Close()

		c.Context().SetContentType(mediaTypePlainText)
		_, err = io.Copy(c.Response().BodyWriter(), rc)
		return err
	}
}

func (s *TestkubeAPI) AbortTestWorkflowExecutionHandler() fiber.Handler {
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

		// Abort the Test Workflow
		err = s.ExecutionWorkerClient.Abort(ctx, execution.Id, executionworkertypes.DestroyOptions{
			Namespace: execution.Namespace,
		})
		if err != nil {
			return s.ClientError(c, "aborting test workflow execution", err)
		}
		s.Metrics.IncAbortTestWorkflow()

		c.Status(http.StatusNoContent)

		return nil
	}
}

func (s *TestkubeAPI) PauseTestWorkflowExecutionHandler() fiber.Handler {
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

		// Pausing the execution
		err = s.ExecutionWorkerClient.Pause(ctx, execution.Id, executionworkertypes.ControlOptions{
			Namespace: execution.Namespace,
		})
		if err != nil {
			return s.ClientError(c, "pausing test workflow execution", err)
		}

		c.Status(http.StatusNoContent)

		return nil
	}
}

func (s *TestkubeAPI) ResumeTestWorkflowExecutionHandler() fiber.Handler {
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

		// Resuming the execution
		err = s.ExecutionWorkerClient.Resume(ctx, execution.Id, executionworkertypes.ControlOptions{
			Namespace: execution.Namespace,
		})
		if err != nil {
			return s.ClientError(c, "resuming test workflow execution", err)
		}

		c.Status(http.StatusNoContent)

		return nil
	}
}

func (s *TestkubeAPI) AbortAllTestWorkflowExecutionsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := c.Context()
		name := c.Params("id")
		errPrefix := fmt.Sprintf("failed to abort test workflow executions '%s'", name)

		// Fetch executions
		filter := testworkflow2.NewExecutionsFilter().WithName(name).WithStatus(string(testkube.RUNNING_TestWorkflowStatus))
		executions, err := s.TestWorkflowResults.GetExecutions(ctx, filter)
		if err != nil {
			if apiutils.IsNotFound(err) {
				c.Status(http.StatusNoContent)
				return nil
			}
			return s.ClientError(c, errPrefix, err)
		}

		for _, execution := range executions {
			err = s.ExecutionWorkerClient.Abort(ctx, execution.Id, executionworkertypes.DestroyOptions{
				Namespace: execution.Namespace,
			})
			if err != nil {
				return s.ClientError(c, errPrefix, err)
			}
			s.Metrics.IncAbortTestWorkflow()
		}

		c.Status(http.StatusNoContent)

		return nil
	}
}

func (s *TestkubeAPI) ListTestWorkflowExecutionArtifactsHandler() fiber.Handler {
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

func (s *TestkubeAPI) GetTestWorkflowArtifactHandler() fiber.Handler {
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

func (s *TestkubeAPI) GetTestWorkflowArtifactArchiveHandler() fiber.Handler {
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

		archive, err := s.ArtifactsStorage.DownloadArchive(c.Context(), execution.Id, "", "", execution.Workflow.Name, values["mask"])
		if err != nil {
			return s.InternalError(c, errPrefix, "could not download workflow artifact archive", err)
		}

		return c.SendStream(archive)
	}
}

func getWorkflowExecutionsFilterFromRequest(c *fiber.Ctx) testworkflow2.Filter {
	filter := testworkflow2.NewExecutionsFilter()
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

	tagSelector := c.Query("tagSelector")
	if tagSelector != "" {
		filter = filter.WithTagSelector(tagSelector)
	}

	actorName := c.Query("actorName")
	if actorName != "" {
		filter = filter.WithActorName(actorName)
	}

	actorType := c.Query("actorType")
	if actorType != "" {
		filter = filter.WithActorType(testkube.TestWorkflowRunningContextActorType(actorType))
	}

	return filter
}
