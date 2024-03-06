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
	"encoding/json"
	"fmt"

	"github.com/gofiber/fiber/v2"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/testworkflowcontroller"
)

func (s *apiTCL) StreamTestWorkflowExecutionNotificationsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := c.Context()
		id := c.Params("id")
		errPrefix := fmt.Sprintf("failed to stream test workflow execution notifications '%s'", id)

		// TODO: Fetch execution from database
		execution := testkube.TestWorkflowExecution{
			Id: id,
		}

		// Check for the logs TODO: Load from the database if possible
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
