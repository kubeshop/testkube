// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package env

import (
	"context"
	"fmt"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/agent"
	"github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/cloud"
	cloudexecutor "github.com/kubeshop/testkube/pkg/cloud/data/executor"
	phttp "github.com/kubeshop/testkube/pkg/http"
	"github.com/kubeshop/testkube/pkg/k8sclient"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/storage/minio"
	"github.com/kubeshop/testkube/pkg/ui"
)

func KubernetesConfig() *rest.Config {
	c, err := rest.InClusterConfig()
	if err != nil {
		var fsErr error
		c, fsErr = k8sclient.GetK8sClientConfig()
		if fsErr != nil {
			ui.Fail(fmt.Errorf("couldn't find Kubernetes config: %w and %w", err, fsErr))
		}
	}
	return c
}

func Kubernetes() *kubernetes.Clientset {
	c, err := kubernetes.NewForConfig(KubernetesConfig())
	if err != nil {
		ui.Fail(fmt.Errorf("couldn't instantiate Kubernetes client: %w", err))
	}
	return c
}

func Testkube() client.Client {
	if UseProxy() {
		return client.NewProxyAPIClient(Kubernetes(), client.NewAPIConfig(Namespace(), config.APIServerName, config.APIServerPort))
	}
	httpClient := phttp.NewClient(true)
	sseClient := phttp.NewSSEClient(true)
	return client.NewDirectAPIClient(httpClient, sseClient, fmt.Sprintf("http://%s:%d", config.APIServerName, config.APIServerPort), "")
}

func ObjectStorageClient() (*minio.Client, error) {
	cfg := Config().ObjectStorage
	opts := minio.GetTLSOptions(cfg.Ssl, cfg.SkipVerify, cfg.CertFile, cfg.KeyFile, cfg.CAFile)
	c := minio.NewClient(cfg.Endpoint, cfg.AccessKeyID, cfg.SecretAccessKey, cfg.Region, cfg.Token, cfg.Bucket, opts...)
	return c, c.Connect()
}

func Cloud(ctx context.Context) cloudexecutor.Executor {
	cfg := Config().Cloud
	grpcConn, err := agent.NewGRPCConnection(ctx, cfg.TlsInsecure, cfg.SkipVerify, cfg.Url, "", "", "", log.DefaultLogger)
	if err != nil {
		ui.Fail(fmt.Errorf("failed to connect with Cloud: %w", err))
	}
	grpcClient := cloud.NewTestKubeCloudAPIClient(grpcConn)
	return cloudexecutor.NewCloudGRPCExecutor(grpcClient, grpcConn, cfg.ApiKey)
}
