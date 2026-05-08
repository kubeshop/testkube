// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//     https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package testworkflowresolver

import (
	"fmt"
	"strconv"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/pkg/expressions"
)

var configFinalizer = expressions.PrefixMachine("config.", expressions.FinalizerFail)

func castParameter(value intstr.IntOrString, schema testworkflowsv1.ParameterSchema) (expressions.Expression, error) {
	v := value.StrVal
	if value.Type == intstr.Int {
		v = strconv.Itoa(int(value.IntVal))
	}
	expr, err := expressions.CompileTemplate(v)
	if err != nil {
		return nil, err
	}
	switch schema.Type {
	case testworkflowsv1.ParameterTypeBoolean:
		return expressions.CastToBool(expr).Resolve()
	case testworkflowsv1.ParameterTypeInteger:
		return expressions.CastToInt(expr).Resolve()
	case testworkflowsv1.ParameterTypeNumber:
		return expressions.CastToFloat(expr).Resolve()
	}
	return expressions.CastToString(expr).Resolve()
}

func createConfigMachine(cfg map[string]intstr.IntOrString, schema map[string]testworkflowsv1.ParameterSchema,
	externalize func(key, value string) (expressions.Expression, error)) (expressions.Machine, error) {
	machine := expressions.NewMachine()
	for k, v := range cfg {
		expr, err := castParameter(v, schema[k])
		if err != nil {
			return nil, errors.Wrap(err, "config."+k)
		}
		if schema[k].Sensitive && externalize != nil {
			expr, err = externalize(k, expr.Template())
			if err != nil {
				return nil, err
			}
		}
		machine.Register("config."+k, expr)
	}
	for k := range schema {
		if schema[k].Default != nil {
			expr, err := castParameter(*schema[k].Default, schema[k])
			if err != nil {
				return nil, errors.Wrap(err, "config."+k)
			}
			if schema[k].Sensitive && externalize != nil {
				expr, err = externalize(k, expr.Template())
				if err != nil {
					return nil, err
				}
			}
			machine.Register("config."+k, expr)
		}
	}
	return machine, nil
}

func EnvVarSourceToSecretExpression(fn func(key, value string) (*corev1.EnvVarSource, error)) func(key, value string) (expressions.Expression, error) {
	return func(key, value string) (expressions.Expression, error) {
		envVar, err := fn(key, value)
		if err != nil {
			return nil, err
		}
		if envVar.SecretKeyRef != nil {
			return expressions.Compile(fmt.Sprintf("secret(%s,%s,true)",
				expressions.NewStringValue(envVar.SecretKeyRef.Name).String(),
				expressions.NewStringValue(envVar.SecretKeyRef.Key).String()))
		}
		return nil, nil
	}
}

func ApplyWorkflowConfig(t *testworkflowsv1.TestWorkflow, cfg map[string]intstr.IntOrString,
	externalize func(key, value string) (expressions.Expression, error)) (*testworkflowsv1.TestWorkflow, error) {
	if t == nil {
		return t, nil
	}
	machine, err := createConfigMachine(cfg, t.Spec.Config, externalize)
	if err != nil {
		return nil, err
	}
	err = expressions.Simplify(&t, machine, configFinalizer)
	return t, err
}

func ApplyWorkflowTemplateConfig(t *testworkflowsv1.TestWorkflowTemplate, cfg map[string]intstr.IntOrString,
	externalize func(key, value string) (expressions.Expression, error)) (*testworkflowsv1.TestWorkflowTemplate, error) {
	if t == nil {
		return t, nil
	}
	machine, err := createConfigMachine(cfg, t.Spec.Config, externalize)
	if err != nil {
		return nil, err
	}
	err = expressions.Simplify(&t, machine, configFinalizer)
	return t, err
}
