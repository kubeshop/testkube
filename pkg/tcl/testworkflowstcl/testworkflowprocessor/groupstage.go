// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package testworkflowprocessor

import (
	"maps"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/pkg/tcl/expressionstcl"
)

type groupStage struct {
	stageMetadata  `expr:"include"`
	stageLifecycle `expr:"include"`
	children       []Stage `expr:"include"`
}

type GroupStage interface {
	Stage
	Children() []Stage
	RecursiveChildren() []Stage
	Add(stages ...Stage) GroupStage
}

func NewGroupStage(ref string) GroupStage {
	return &groupStage{
		stageMetadata: stageMetadata{ref: ref},
	}
}

func (s *groupStage) Len() int {
	count := 0
	for _, ch := range s.children {
		count += ch.Len()
	}
	return count
}

func (s *groupStage) Signature() Signature {
	sig := []Signature(nil)
	for _, ch := range s.children {
		si := ch.Signature()
		_, ok := ch.(GroupStage)
		// Include children directly, if the stage is virtual
		if ok && si.Name() == "" && !si.Optional() && !si.Negative() {
			sig = append(sig, si.Children()...)
		} else {
			sig = append(sig, si)
		}
	}

	return &signature{
		RefValue:      s.ref,
		NameValue:     s.name,
		OptionalValue: s.optional,
		NegativeValue: s.negative,
		ChildrenValue: sig,
	}
}

func (s *groupStage) ContainerStages() []ContainerStage {
	c := []ContainerStage(nil)
	for _, ch := range s.children {
		c = append(c, ch.ContainerStages()...)
	}
	return c
}

func (s *groupStage) Children() []Stage {
	return s.children
}

func (s *groupStage) RecursiveChildren() []Stage {
	res := make([]Stage, 0)
	for _, ch := range s.children {
		if v, ok := ch.(GroupStage); ok {
			res = append(res, v.RecursiveChildren()...)
		} else {
			res = append(res, ch)
		}
	}
	return res
}

func (s *groupStage) GetImages() map[string]struct{} {
	v := make(map[string]struct{})
	for _, ch := range s.children {
		maps.Copy(v, ch.GetImages())
	}
	return v
}

func (s *groupStage) Flatten() []Stage {
	// Flatten children
	next := []Stage(nil)
	for _, ch := range s.children {
		next = append(next, ch.Flatten()...)
	}
	s.children = next

	// Delete empty stage
	if len(s.children) == 0 {
		return nil
	}

	// Merge stage into single one below if possible
	first := s.children[0]
	if len(s.children) == 1 && (s.name == "" || first.Name() == "") {
		first.SetName(first.Name(), s.name)
		first.AppendConditions(s.condition)
		if s.negative {
			first.SetNegative(!first.Negative())
		}
		if s.optional {
			first.SetOptional(true)
		}
		return []Stage{first}
	}

	return []Stage{s}
}

func (s *groupStage) Add(stages ...Stage) GroupStage {
	for _, ch := range stages {
		s.children = append(s.children, ch.Flatten()...)
	}
	return s
}

func (s *groupStage) Resolve(m ...expressionstcl.Machine) error {
	for _, ch := range s.children {
		err := ch.Resolve(m...)
		if err != nil {
			return errors.Wrap(err, "group stage container")
		}
	}
	return expressionstcl.Simplify(s.stageMetadata, m...)
}
