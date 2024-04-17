// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package spawn

import (
	"errors"
	"fmt"
	"math"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	common2 "github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/common"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/tcl/expressionstcl"
)

func FromInstruction(name string, instruction testworkflowsv1.SpawnInstructionBase, machines ...expressionstcl.Machine) (Service, error) {
	// Validate the instruction
	if len(instruction.Pod.Spec.Containers) == 0 {
		return Service{}, errors.New("pod.spec.containers: no containers provided")
	}

	// Resolve the shards and matrix
	params, err := common2.GetParamsSpec(instruction.Matrix, instruction.Shards, instruction.Count, instruction.MaxCount, machines...)
	if err != nil {
		return Service{}, fmt.Errorf("parsing spec: %w", err)
	}

	// Compute parallelism
	var parallelism *int64
	if instruction.Parallelism != nil {
		parallelismVal, err := readCount(*instruction.Parallelism, machines...)
		if err != nil {
			return Service{}, fmt.Errorf("parallelism: %w", err)
		}
		parallelism = &parallelismVal
	}
	if parallelism == nil {
		parallelism = common.Ptr(int64(math.MaxInt64))
	}
	if *parallelism > params.Count {
		parallelism = common.Ptr(params.Count)
	}

	// Build the service
	var pod corev1.PodTemplateSpec
	if instruction.Pod != nil {
		pod = *instruction.Pod
	}
	svc := Service{
		Name:        name,
		Description: instruction.Description,
		Strategy:    instruction.Strategy,
		Count:       params.ShardCount,
		Parallelism: *parallelism,
		Logs:        common.ResolvePtr(instruction.Logs, false),
		Timeout:     instruction.Timeout,
		Matrix:      params.Matrix,
		Shards:      params.Shards,
		Ready:       instruction.Ready,
		Error:       instruction.Error,
		PodTemplate: pod,
		Files:       instruction.Files,
		Transfer:    instruction.Transfer,
	}

	// Define the default success/error clauses
	if svc.Ready == "" {
		svc.Ready = "success"
	}
	if svc.Error == "" {
		svc.Error = "deleted || failed"
	}

	// Save the service
	return svc, nil
}

func readCount(s intstr.IntOrString, machines ...expressionstcl.Machine) (int64, error) {
	countExpr, err := expressionstcl.CompileAndResolve(s.String(), machines...)
	if err != nil {
		return 0, fmt.Errorf("%s: invalid: %s", s.String(), err)
	}
	if countExpr.Static() == nil {
		return 0, fmt.Errorf("%s: could not resolve: %s", s.String(), err)
	}
	countVal, err := countExpr.Static().IntValue()
	if err != nil {
		return 0, fmt.Errorf("%s: could not convert to int: %s", s.String(), err)
	}
	if countVal < 0 {
		return 0, fmt.Errorf("%s: should not be lower than zero", s.String())
	}
	return countVal, nil
}
