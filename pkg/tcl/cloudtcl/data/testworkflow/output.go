// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package testworkflow

import (
	"bytes"
	"context"
	"io"
	"net/http"

	"github.com/kubeshop/testkube/pkg/tcl/repositorytcl/testworkflow"

	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/cloud/data/executor"
)

var _ testworkflow.OutputRepository = (*CloudOutputRepository)(nil)

type CloudOutputRepository struct {
	executor executor.Executor
}

func NewCloudOutputRepository(client cloud.TestKubeCloudAPIClient, grpcConn *grpc.ClientConn, apiKey string) *CloudOutputRepository {
	return &CloudOutputRepository{executor: executor.NewCloudGRPCExecutor(client, grpcConn, apiKey)}
}

// PresignSaveLog builds presigned storage URL to save the output in Cloud
func (r *CloudOutputRepository) PresignSaveLog(ctx context.Context, id, workflowName string) (string, error) {
	req := OutputPresignSaveLogRequest{ID: id, WorkflowName: workflowName}
	process := func(v OutputPresignSaveLogResponse) string {
		return v.URL
	}
	return pass(r.executor, ctx, req, process)
}

// PresignReadLog builds presigned storage URL to read the output from Cloud
func (r *CloudOutputRepository) PresignReadLog(ctx context.Context, id, workflowName string) (string, error) {
	req := OutputPresignReadLogRequest{ID: id, WorkflowName: workflowName}
	process := func(v OutputPresignReadLogResponse) string {
		return v.URL
	}
	return pass(r.executor, ctx, req, process)
}

// SaveLog streams the output from the workflow to Cloud
func (r *CloudOutputRepository) SaveLog(ctx context.Context, id, workflowName string, reader io.Reader) error {
	url, err := r.PresignSaveLog(ctx, id, workflowName)
	if err != nil {
		return err
	}
	// FIXME: It should stream instead
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewBuffer(data))
	req.Header.Add("Content-Type", "application/octet-stream")
	if err != nil {
		return err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to save file in cloud storage")
	}
	if res.StatusCode != http.StatusOK {
		return errors.Errorf("error saving file with presigned url: expected 200 OK response code, got %d", res.StatusCode)
	}
	return nil
}

// ReadLog streams the output from Cloud
func (r *CloudOutputRepository) ReadLog(ctx context.Context, id, workflowName string) (io.Reader, error) {
	url, err := r.PresignReadLog(ctx, id, workflowName)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get file from cloud storage")
	}
	if res.StatusCode != http.StatusOK {
		return nil, errors.Errorf("error getting file from presigned url: expected 200 OK response code, got %d", res.StatusCode)
	}
	return res.Body, nil
}

// HasLog checks if there is an output in Cloud
func (r *CloudOutputRepository) HasLog(ctx context.Context, id, workflowName string) (bool, error) {
	req := OutputHasLogRequest{ID: id, WorkflowName: workflowName}
	process := func(v OutputHasLogResponse) bool {
		return v.Has
	}
	return pass(r.executor, ctx, req, process)
}
