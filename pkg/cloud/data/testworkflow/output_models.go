// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package testworkflow

type OutputPresignSaveLogRequest struct {
	ID           string `json:"id"`
	WorkflowName string `json:"workflowName"`
}

type OutputPresignSaveLogResponse struct {
	URL string `json:"url"`
}

type OutputPresignReadLogRequest struct {
	ID           string `json:"id"`
	WorkflowName string `json:"workflowName"`
}

type OutputPresignReadLogResponse struct {
	URL string `json:"url"`
}

type OutputHasLogRequest struct {
	ID           string `json:"id"`
	WorkflowName string `json:"workflowName"`
}

type OutputHasLogResponse struct {
	Has bool `json:"has"`
}
