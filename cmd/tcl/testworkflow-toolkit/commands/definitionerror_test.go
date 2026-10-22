// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package commands

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/spawn"
	toolkitcommon "github.com/kubeshop/testkube/cmd/testworkflow-toolkit/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func TestDefinitionError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantText   string
		wantReason testkube.StopReason
	}{
		{
			name:       "an expression that cannot resolve names the part of the workflow",
			err:        &DefinitionError{Subject: `The service "slave"`, Err: errors.New("count: env.SLAVES_COUNT: could not resolve")},
			wantText:   `The service "slave" is invalid: count: env.SLAVES_COUNT: could not resolve.`,
			wantReason: testkube.StopReason(testkube.StartReasonDefinitionInvalid),
		},
		{
			name:       "a wrapped error in the definition keeps its code",
			err:        fmt.Errorf("services: %w", &DefinitionError{Subject: "The parallel step", Err: errors.New("no spec provided.")}),
			wantText:   "services: The parallel step is invalid: no spec provided.",
			wantReason: testkube.StopReason(testkube.StartReasonDefinitionInvalid),
		},
		{
			name:       "a transfer path that cannot resolve is an error in the definition",
			err:        definitionOrRuntime(`The service "db"`, fmt.Errorf("transfer: %w", &spawn.ExpressionError{Err: errors.New("0.from: env.DIR: could not resolve")})),
			wantText:   `The service "db" is invalid: transfer: 0.from: env.DIR: could not resolve.`,
			wantReason: testkube.StopReason(testkube.StartReasonDefinitionInvalid),
		},
		{
			name:     "a transfer that cannot pack its files fails at run time",
			err:      definitionOrRuntime(`The service "db"`, fmt.Errorf("transfer: %w", errors.New("0: no space left on device"))),
			wantText: `The service "db": transfer: 0: no space left on device`,
		},
		{
			name:     "an error at run time gets no code",
			err:      errors.New("starting transfer server: address in use"),
			wantText: "starting transfer server: address in use",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantText, tt.err.Error())
			assert.Equal(t, tt.wantReason, toolkitcommon.ReasonOf(tt.err))
		})
	}
}
