// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//     https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package testworkflowresolver

import (
	"strconv"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/intstr"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/pkg/tcl/expressionstcl"
)

var configFinalizer = expressionstcl.PrefixMachine("config.", expressionstcl.FinalizerFail)

func castParameter(value intstr.IntOrString, schema testworkflowsv1.ParameterSchema) (expressionstcl.Expression, error) {
	v := value.StrVal
	if value.Type == intstr.Int {
		v = strconv.Itoa(int(value.IntVal))
	}
	expr, err := expressionstcl.CompileTemplate(v)
	if err != nil {
		return nil, err
	}
	switch schema.Type {
	case testworkflowsv1.ParameterTypeBoolean:
		return expressionstcl.CastToBool(expr).Resolve()
	case testworkflowsv1.ParameterTypeInteger:
		return expressionstcl.CastToInt(expr).Resolve()
	case testworkflowsv1.ParameterTypeNumber:
		return expressionstcl.CastToFloat(expr).Resolve()
	}
	return expressionstcl.CastToString(expr).Resolve()
}

func createConfigMachine(cfg map[string]intstr.IntOrString, schema map[string]testworkflowsv1.ParameterSchema) (expressionstcl.Machine, error) {
	machine := expressionstcl.NewMachine()
	for k, v := range cfg {
		expr, err := castParameter(v, schema[k])
		if err != nil {
			return nil, errors.Wrap(err, "config."+k)
		}
		machine.Register("config."+k, expr)
	}
	for k := range schema {
		if schema[k].Default != nil {
			expr, err := castParameter(*schema[k].Default, schema[k])
			if err != nil {
				return nil, errors.Wrap(err, "config."+k)
			}
			machine.Register("config."+k, expr)
		}
	}
	return machine, nil
}

func ApplyWorkflowConfig(t *testworkflowsv1.TestWorkflow, cfg map[string]intstr.IntOrString) (*testworkflowsv1.TestWorkflow, error) {
	if t == nil {
		return t, nil
	}
	machine, err := createConfigMachine(cfg, t.Spec.Config)
	if err != nil {
		return nil, err
	}
	err = expressionstcl.Simplify(&t, machine, configFinalizer)
	return t, err
}

func ApplyWorkflowTemplateConfig(t *testworkflowsv1.TestWorkflowTemplate, cfg map[string]intstr.IntOrString) (*testworkflowsv1.TestWorkflowTemplate, error) {
	if t == nil {
		return t, nil
	}
	machine, err := createConfigMachine(cfg, t.Spec.Config)
	if err != nil {
		return nil, err
	}
	err = expressionstcl.Simplify(&t, machine, configFinalizer)
	return t, err
}
