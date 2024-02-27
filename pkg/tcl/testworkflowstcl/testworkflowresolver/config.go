// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//     https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package testworkflowresolver

import (
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/intstr"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/pkg/tcl/expressionstcl"
)

var configFinalizer = expressionstcl.PrefixMachine("config.", expressionstcl.FinalizerFail)

func createConfigMachine(cfg map[string]intstr.IntOrString) (expressionstcl.Machine, error) {
	machine := expressionstcl.NewMachine()
	for k, v := range cfg {
		if v.Type == intstr.String {
			expr, err := expressionstcl.CompileTemplate(v.StrVal)
			if err != nil {
				return nil, errors.Wrap(err, "config."+k)
			}
			machine.Register("config."+k, expr)
		} else {
			machine.Register("config."+k, v.IntVal)
		}
	}
	return machine, nil
}

func ApplyWorkflowConfig(t *testworkflowsv1.TestWorkflow, cfg map[string]intstr.IntOrString) (*testworkflowsv1.TestWorkflow, error) {
	if t == nil {
		return t, nil
	}
	machine, err := createConfigMachine(cfg)
	if err != nil {
		return nil, err
	}
	err = expressionstcl.SimplifyStruct(&t, machine, configFinalizer)
	return t, err
}

func ApplyWorkflowTemplateConfig(t *testworkflowsv1.TestWorkflowTemplate, cfg map[string]intstr.IntOrString) (*testworkflowsv1.TestWorkflowTemplate, error) {
	if t == nil {
		return t, nil
	}
	machine, err := createConfigMachine(cfg)
	if err != nil {
		return nil, err
	}
	err = expressionstcl.SimplifyStruct(&t, machine, configFinalizer)
	return t, err
}
