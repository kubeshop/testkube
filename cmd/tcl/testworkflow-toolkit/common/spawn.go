// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package common

import (
	"encoding/json"

	"github.com/pkg/errors"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/transfer"
	"github.com/kubeshop/testkube/pkg/tcl/expressionstcl"
)

func ProcessTransfer(transferSrv transfer.Server, transfer []testworkflowsv1.StepParallelTransfer, machines ...expressionstcl.Machine) ([]testworkflowsv1.ContentTarball, error) {
	if len(transfer) == 0 {
		return nil, nil
	}
	result := make([]testworkflowsv1.ContentTarball, 0, len(transfer))
	for ti, t := range transfer {
		// Parse 'from' clause
		from, err := expressionstcl.EvalTemplate(t.From, machines...)
		if err != nil {
			return nil, errors.Wrapf(err, "transfer.%d.from", ti)
		}

		// Parse 'to' clause
		to := from
		if t.To != "" {
			to, err = expressionstcl.EvalTemplate(t.To, machines...)
			if err != nil {
				return nil, errors.Wrapf(err, "transfer.%d.to", ti)
			}
		}

		// Parse 'files' clause
		patterns := []string{"**/*"}
		if t.Files != nil && !t.Files.Dynamic {
			patterns = t.Files.Static
		} else if t.Files != nil && t.Files.Dynamic {
			patternsExpr, err := expressionstcl.EvalExpression(t.Files.Expression, machines...)
			if err != nil {
				return nil, errors.Wrapf(err, "transfer.%d.files", ti)
			}
			patternsList, err := patternsExpr.Static().SliceValue()
			if err != nil {
				return nil, errors.Wrapf(err, "transfer.%d.files", ti)
			}
			patterns = make([]string, len(patternsList))
			for pi, p := range patternsList {
				if s, ok := p.(string); ok {
					patterns[pi] = s
				} else {
					p, err := json.Marshal(s)
					if err != nil {
						return nil, errors.Wrapf(err, "transfer.%d.files.%d", ti, pi)
					}
					patterns[pi] = string(p)
				}
			}
		}

		entry, err := transferSrv.Include(from, patterns)
		if err != nil {
			return nil, errors.Wrapf(err, "transfer.%d", ti)
		}
		result = append(result, testworkflowsv1.ContentTarball{Url: entry.Url, Path: to, Mount: t.Mount})
	}
	return result, nil
}
