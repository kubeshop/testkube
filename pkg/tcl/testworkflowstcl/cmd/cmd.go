// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package cmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
)

func GetRunningContext(runContext, token string, interfaceType testkube.TestWorkflowRunningContextInterfaceType) *testkube.TestWorkflowRunningContext {
	var name, email string
	if token != "" {
		payload, err := getJWTPayload(token)
		if err == nil {
			if value, ok := payload["name"]; ok {
				name = fmt.Sprint(value)
			}

			if value, ok := payload["email"]; ok {
				email = fmt.Sprint(value)
			}
		}
	}

	return &testkube.TestWorkflowRunningContext{
		Interface_: &testkube.TestWorkflowRunningContextInterface{
			Name:  runContext,
			Type_: common.Ptr(interfaceType),
		},
		Actor: &testkube.TestWorkflowRunningContextActor{
			Type_: common.Ptr(testkube.USER_TestWorkflowRunningContextActorType),
			Name:  name,
			Email: email,
		},
	}
}

func getJWTPayload(token string) (map[string]interface{}, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token format")
	}

	// Decode the payload
	payloadJSON, err := base64.RawURLEncoding.DecodeString((parts[1]))
	if err != nil {
		return nil, fmt.Errorf("failed to decode payload: %v", err)
	}

	// Unmarshal the payload into maps
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
		return nil, fmt.Errorf("failed to parse payload JSON: %v", err)
	}

	return payload, nil
}

func PrintRunningContext(ui *ui.UI, execution testkube.TestWorkflowExecution) {
	if execution.RunningContext != nil {
		ui.Warn("Running context:     ")
		ctx := execution.RunningContext
		if ctx.Interface_ != nil {
			ui.Warn("Interface:             ")
			if ctx.Interface_.Name != "" {
				ui.Warn("  Name:                ", ctx.Interface_.Name)
			}
			if ctx.Interface_.Type_ != nil {
				ui.Warn("  Type:                ", string(*ctx.Interface_.Type_))
			}
		}
		if ctx.Actor != nil {
			ui.Warn("Actor:                 ")
			fields := []struct {
				name  string
				value string
			}{
				{
					"  Name:                ",
					ctx.Actor.Name,
				},
				{
					"  Email:               ",
					ctx.Actor.Email,
				},
				{
					"  Execution id:        ",
					ctx.Actor.ExecutionId,
				},
				{
					"  Execution path:      ",
					ctx.Actor.ExecutionPath,
				},
				{
					"  Execution reference: ",
					ctx.Actor.ExecutionReference,
				},
			}

			for _, field := range fields {
				if field.value != "" {
					ui.Warn(field.name, field.value)
				}
			}
			if ctx.Actor.Type_ != nil {
				ui.Warn("  Type:                ", string(*ctx.Actor.Type_))
			}
		}
	}
}
