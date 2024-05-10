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
	if cr.Parallel != nil {
		maps.Copy(v, listSpecTemplates(cr.Parallel.TestWorkflowSpec))
	}
	for i := range cr.Setup {
		maps.Copy(v, listStepTemplates(cr.Setup[i]))
	}
	for i := range cr.Steps {
		maps.Copy(v, listStepTemplates(cr.Steps[i]))
	}
	return v
}

func listSpecTemplates(spec testworkflowsv1.TestWorkflowSpec) map[string]struct{} {
	v := make(map[string]struct{})
	for i := range spec.Use {
		v[GetInternalTemplateName(spec.Use[i].Name)] = struct{}{}
	}
	for i := range spec.Setup {
		maps.Copy(v, listStepTemplates(spec.Setup[i]))
	}
	for i := range spec.Steps {
		maps.Copy(v, listStepTemplates(spec.Steps[i]))
	}
	for i := range spec.After {
		maps.Copy(v, listStepTemplates(spec.After[i]))
	}
	return v
}

func ListTemplates(cr *testworkflowsv1.TestWorkflow) map[string]struct{} {
	if cr == nil {
		return nil
	}
	return listSpecTemplates(cr.Spec)
}
