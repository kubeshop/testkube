package v1

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/valyala/fasthttp"

	"github.com/kubeshop/testkube/pkg/ai"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func (s TestkubeAPI) AnalyzeTestExecutionSync() fiber.Handler {
	ai := ai.NewOpenAI(os.Getenv("OPENAI_KEY"))
	return func(c *fiber.Ctx) error {
		ctx := context.Background()

		execution, err := s.ExecutionResults.Get(ctx, c.Params("executionID"))
		if err != nil {
			return s.Error(c, http.StatusInternalServerError, fmt.Errorf("can't get execution details: %w", err))
		}

		if *execution.ExecutionResult.Status != *testkube.ExecutionStatusFailed {
			return s.Error(c, http.StatusBadRequest, fmt.Errorf("execution status is not failed, I can analyze only failed executions"))
		}

		resp, err := ai.AnalyzeTestExecution(ctx, execution)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, fmt.Errorf("client could not analyze test execution: %w", err))
		}

		c.Status(http.StatusCreated)
		return c.JSON(testkube.AiAnalysisResponse{
			Message: resp,
		})
	}
}

func (s TestkubeAPI) AnalyzeTestExecutionSSE() fiber.Handler {
	ai := ai.NewOpenAI(os.Getenv("OPENAI_KEY"))
	return func(c *fiber.Ctx) error {

		c.Set("Content-Type", "text/event-stream")
		c.Set("Cache-Control", "no-cache")
		c.Set("Connection", "keep-alive")
		c.Set("Transfer-Encoding", "chunked")
		executionId := c.Params("executionID")
		ctx := context.Background()
		l := s.Log.With("executionID", executionId)

		l.Debugw("starting AI result analysis", "executionID", executionId)

		execution, err := s.ExecutionResults.Get(ctx, executionId)
		if err != nil {
			l.Errorw("can't get execution details", "error", err)
			return err
		}

		if *execution.ExecutionResult.Status != *testkube.ExecutionStatusFailed {
			l.Errorf("execution status is not failed, I can analyze only failed executions")
			return err
		}

		c.Context().SetBodyStreamWriter(fasthttp.StreamWriter(func(w *bufio.Writer) {
			stream, err := ai.AnalyzeTestExecutionStream(ctx, execution)
			if err != nil {
				l.Errorf("can't get analysis stream: %v", err)
			}

			for line := range stream {
				l.Debugw("sending log line to websocket", "line", line)
				fmt.Fprintf(w, "data: %s\n\n", line)
				err := w.Flush()
				if err != nil {
					// Refreshing page in web browser will establish a new
					// SSE connection, but only (the last) one is alive, so
					// dead connections must be closed here.
					fmt.Printf("Error while flushing: %v. Closing http connection.\n", err)
					break
				}

			}
			fmt.Fprintf(w, "data: \n\nDONE\n\n")
		}))
		return nil
	}
}

func (s TestkubeAPI) AnalyzeTestExecution() fiber.Handler {
	ai := ai.NewOpenAI(os.Getenv("OPENAI_KEY"))

	return websocket.New(func(c *websocket.Conn) {
		defer c.Conn.Close()

		_ = c.WriteJSON(AIResponseChunk{Message: "Starting AI analysis...\n"})
		_ = c.WriteJSON(AIResponseChunk{Message: "\n"})

		ctx := context.Background()
		executionId := c.Params("executionID")
		l := s.Log.With("executionID", executionId)

		l.Debugw("starting AI result analysis", "executionID", executionId)

		execution, err := s.ExecutionResults.Get(ctx, executionId)
		if err != nil {
			return
		}

		if *execution.ExecutionResult.Status != *testkube.ExecutionStatusFailed {
			l.Errorf("execution status is not failed, I can analyze only failed executions")
		}

		stream, err := ai.AnalyzeTestExecutionStream(ctx, execution)
		if err != nil {
			l.Errorf("can't get analysis stream: %v", err)
		}

		for line := range stream {
			l.Debugw("sending log line to websocket", "line", line)
			_ = c.WriteJSON(AIResponseChunk{Message: line})
		}

		_ = c.WriteJSON(AIResponseChunk{Message: "\n\n"})
		_ = c.WriteJSON(AIResponseChunk{Message: "DONE"})
	})
}

type AIResponseChunk struct {
	Message string `json:"message"`
}
