// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//     https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package testworkflowresolver

import (
	"maps"
	"strings"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
)

func GetInternalTemplateName(name string) string {
	return strings.ReplaceAll(name, "/", "--")
}

func GetDisplayTemplateName(name string) string {
	return strings.ReplaceAll(name, "--", "/")
}

func listStepTemplates(cr testworkflowsv1.Step) map[string]struct{} {
	v := make(map[string]struct{})
	if cr.Template != nil {
		v[GetInternalTemplateName(cr.Template.Name)] = struct{}{}
	}
	for i := range cr.Use {
		v[GetInternalTemplateName(cr.Use[i].Name)] = struct{}{}
	}
	for i := range cr.Setup {
		maps.Copy(v, listStepTemplates(cr.Setup[i]))
	}
	for i := range cr.Steps {
		maps.Copy(v, listStepTemplates(cr.Steps[i]))
	}
	return v
}

func ListTemplates(cr *testworkflowsv1.TestWorkflow) map[string]struct{} {
	if cr == nil {
		return nil
	}
	v := make(map[string]struct{})
	for i := range cr.Spec.Use {
		v[GetInternalTemplateName(cr.Spec.Use[i].Name)] = struct{}{}
	}
	for i := range cr.Spec.Setup {
		maps.Copy(v, listStepTemplates(cr.Spec.Setup[i]))
	}
	for i := range cr.Spec.Steps {
		maps.Copy(v, listStepTemplates(cr.Spec.Steps[i]))
	}
	for i := range cr.Spec.After {
		maps.Copy(v, listStepTemplates(cr.Spec.After[i]))
	}
	return v
}
