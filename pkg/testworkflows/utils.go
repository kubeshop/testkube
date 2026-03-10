// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//     https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package testworkflows

import "github.com/kubeshop/testkube/pkg/api/v1/testkube"

// CountMapBytes returns the total bytes of the map
func CountMapBytes(m map[string]string) int {
	totalBytes := 0
	for k, v := range m {
		totalBytes += len(k) + len(v)
	}
	return totalBytes
}

// FlattenSignatures transform signatures tree into the list
func FlattenSignatures(sig []testkube.TestWorkflowSignature) []testkube.TestWorkflowSignature {
	res := make([]testkube.TestWorkflowSignature, 0)
	for _, s := range sig {
		if len(s.Children) == 0 {
			res = append(res, s)
		} else {
			res = append(res, FlattenSignatures(s.Children)...)
		}
	}
	return res
}

func IsWorkflowSilent(workflow *testkube.TestWorkflow) bool {
	if workflow == nil || workflow.Spec == nil || workflow.Spec.Execution == nil || workflow.Spec.Execution.Silent == nil {
		return false
	}
	return *workflow.Spec.Execution.Silent
}

// NewSilenceAllSilentMode returns a SilentMode that silences all processing.
func NewSilenceAllSilentMode() *testkube.SilentMode {
	return &testkube.SilentMode{
		Webhooks: true,
		Insights: true,
		Health:   true,
		Metrics:  true,
		Cdevents: true,
	}
}
