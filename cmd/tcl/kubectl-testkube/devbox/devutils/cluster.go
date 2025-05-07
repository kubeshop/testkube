// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package devutils

import (
	"strings"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kubeclient "github.com/kubeshop/testkube-operator/pkg/client"
	"github.com/kubeshop/testkube/pkg/k8sclient"
)

type ClusterObject struct {
	cfg         *rest.Config
	clientSet   *kubernetes.Clientset
	kubeClient  client.Client
	versionInfo *version.Info
	forcedOs    string
	forcedArch  string
}

func NewCluster(forcedOs, forcedArchitecture string) (*ClusterObject, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		config, err = k8sclient.GetK8sClientConfig()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get Kubernetes config")
		}
	}
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create Kubernetes client")
	}
	info, err := clientSet.ServerVersion()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get Kubernetes cluster details")
	}
	kubeClient, err := kubeclient.GetClient()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create Kubernetes client wrapper")
	}

	return &ClusterObject{
		clientSet:   clientSet,
		kubeClient:  kubeClient,
		versionInfo: info,
		cfg:         config,
		forcedOs:    forcedOs,
		forcedArch:  forcedArchitecture,
	}, nil
}

func (c *ClusterObject) ClientSet() *kubernetes.Clientset {
	return c.clientSet
}

func (c *ClusterObject) KubeClient() client.Client {
	return c.kubeClient
}

func (c *ClusterObject) Config() *rest.Config {
	return c.cfg
}

func (c *ClusterObject) Namespace(name, executionName string) *NamespaceObject {
	return NewNamespace(c.clientSet, c.cfg, name, executionName)
}

func (c *ClusterObject) Host() string {
	return c.cfg.Host
}

func (c *ClusterObject) OperatingSystem() string {
	if c.forcedOs != "" {
		return c.forcedOs
	}
	return strings.Split(c.versionInfo.Platform, "/")[0]
}

func (c *ClusterObject) Architecture() string {
	if c.forcedArch != "" {
		return c.forcedArch
	}
	return strings.Split(c.versionInfo.Platform, "/")[1]
}
