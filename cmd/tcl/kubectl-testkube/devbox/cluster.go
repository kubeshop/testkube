// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package devbox

import (
	"strings"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/kubeshop/testkube/pkg/k8sclient"
)

type clusterObj struct {
	cfg         *rest.Config
	clientSet   *kubernetes.Clientset
	versionInfo *version.Info
}

func NewCluster() (*clusterObj, error) {
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

	return &clusterObj{
		clientSet:   clientSet,
		versionInfo: info,
		cfg:         config,
	}, nil
}

func (c *clusterObj) Debug() {
	PrintHeader("Cluster")
	PrintItem("Address", c.cfg.Host, "")
	PrintItem("Platform", c.versionInfo.Platform, "")
	PrintItem("Version", c.versionInfo.GitVersion, "")
}

func (c *clusterObj) ClientSet() *kubernetes.Clientset {
	return c.clientSet
}

func (c *clusterObj) Config() *rest.Config {
	return c.cfg
}

func (c *clusterObj) Namespace(name string) *namespaceObj {
	return NewNamespace(c.clientSet, name)
}

func (c *clusterObj) ImageRegistry(namespace string) *imageRegistryObj {
	return NewImageRegistry(c.clientSet, c.cfg, namespace)
}

func (c *clusterObj) ObjectStorage(namespace string) *objectStorageObj {
	return NewObjectStorage(c.clientSet, c.cfg, namespace)
}

func (c *clusterObj) PodInterceptor(namespace string) *podInterceptorObj {
	return NewPodInterceptor(c.clientSet, c.cfg, namespace)
}

func (c *clusterObj) OperatingSystem() string {
	return strings.Split(c.versionInfo.Platform, "/")[0]
}

func (c *clusterObj) Architecture() string {
	return strings.Split(c.versionInfo.Platform, "/")[1]
}
