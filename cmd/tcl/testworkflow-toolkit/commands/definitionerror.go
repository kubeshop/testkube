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

	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/spawn"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// DefinitionError is an error in the definition of a toolkit step, for example an expression that
// cannot resolve. The user fixes it in the workflow, so it is a configuration error and not a
// failure of the test.
type DefinitionError struct {
	// Subject names the part of the workflow that holds the error, for example `The service "db"`.
	Subject string
	Err     error
}

func (e *DefinitionError) Error() string {
	return fmt.Sprintf("%s is invalid: %s", e.Subject, endSentence(e.Err.Error()))
}

func (e *DefinitionError) Unwrap() error {
	return e.Err
}

// Reason returns the code of an error in the definition of a step.
func (e *DefinitionError) Reason() testkube.StopReason {
	return definitionInvalid
}

// definitionOrRuntime marks the error as an error in the definition of the subject when an
// expression caused it. Other errors, for example a full disk while a transfer packs its files, are
// failures at run time, so they only name the subject.
func definitionOrRuntime(subject string, err error) error {
	var exprErr *spawn.ExpressionError
	if errors.As(err, &exprErr) {
		return &DefinitionError{Subject: subject, Err: err}
	}
	return fmt.Errorf("%s: %w", subject, err)
}

// definitionInvalid is the code of an error in the definition of a step.
const definitionInvalid = testkube.StopReason(testkube.StartReasonDefinitionInvalid)
