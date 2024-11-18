// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package devutils

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	testworkflowsclientv1 "github.com/kubeshop/testkube-operator/pkg/client/testworkflows/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/mapper/testworkflows"
)

type ResourcesClient interface {
	CreateTestWorkflow(workflow testkube.TestWorkflow) (testkube.TestWorkflow, error)
	UpdateTestWorkflow(workflow testkube.TestWorkflow) (testkube.TestWorkflow, error)
	DeleteTestWorkflow(name string) error

	CreateTestWorkflowTemplate(workflow testkube.TestWorkflowTemplate) (testkube.TestWorkflowTemplate, error)
	UpdateTestWorkflowTemplate(workflow testkube.TestWorkflowTemplate) (testkube.TestWorkflowTemplate, error)
	DeleteTestWorkflowTemplate(name string) error
}

type ossResourcesClient struct {
	namespace             string
	testWorkflows         *testworkflowsclientv1.TestWorkflowsClient
	testWorkflowTemplates *testworkflowsclientv1.TestWorkflowTemplatesClient
}

func NewDirectResourcesClient(kubeClient client.Client, namespace string) ResourcesClient {
	return &ossResourcesClient{
		namespace:             namespace,
		testWorkflows:         testworkflowsclientv1.NewClient(kubeClient, namespace),
		testWorkflowTemplates: testworkflowsclientv1.NewTestWorkflowTemplatesClient(kubeClient, namespace),
	}
}

func (r *ossResourcesClient) CreateTestWorkflow(workflow testkube.TestWorkflow) (result testkube.TestWorkflow, err error) {
	workflow.Namespace = r.namespace
	v, err := r.testWorkflows.Create(testworkflows.MapAPIToKube(&workflow))
	if err != nil {
		return
	}
	return *testworkflows.MapKubeToAPI(v), nil
}

func (r *ossResourcesClient) UpdateTestWorkflow(workflow testkube.TestWorkflow) (result testkube.TestWorkflow, err error) {
	prev, err := r.testWorkflows.Get(workflow.Name)
	if err != nil {
		return r.CreateTestWorkflow(workflow)
	}
	cr := testworkflows.MapAPIToKube(&workflow)
	cr.Namespace = r.namespace
	cr.ResourceVersion = prev.ResourceVersion
	v, err := r.testWorkflows.Update(cr)
	if err != nil {
		return
	}
	return *testworkflows.MapKubeToAPI(v), nil
}

func (r *ossResourcesClient) DeleteTestWorkflow(name string) error {
	return r.testWorkflows.Delete(name)
}

func (r *ossResourcesClient) CreateTestWorkflowTemplate(template testkube.TestWorkflowTemplate) (result testkube.TestWorkflowTemplate, err error) {
	template.Namespace = r.namespace
	v, err := r.testWorkflowTemplates.Create(testworkflows.MapTemplateAPIToKube(&template))
	if err != nil {
		return
	}
	return *testworkflows.MapTemplateKubeToAPI(v), nil
}

func (r *ossResourcesClient) UpdateTestWorkflowTemplate(template testkube.TestWorkflowTemplate) (result testkube.TestWorkflowTemplate, err error) {
	prev, err := r.testWorkflowTemplates.Get(template.Name)
	if err != nil {
		return r.CreateTestWorkflowTemplate(template)
	}
	cr := testworkflows.MapTemplateAPIToKube(&template)
	cr.Namespace = r.namespace
	cr.ResourceVersion = prev.ResourceVersion
	v, err := r.testWorkflowTemplates.Update(cr)
	if err != nil {
		return
	}
	return *testworkflows.MapTemplateKubeToAPI(v), nil
}

func (r *ossResourcesClient) DeleteTestWorkflowTemplate(name string) error {
	return r.testWorkflowTemplates.Delete(name)
}
