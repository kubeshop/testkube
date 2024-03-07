// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package testworkflowprocessor

type StageMetadata interface {
	Ref() string
	Name() string
	Category() string

	SetName(name string) StageMetadata
	SetCategory(category string) StageMetadata
}

type stageMetadata struct {
	ref      string
	name     string
	category string
}

func NewStageMetadata(ref string) StageMetadata {
	return &stageMetadata{ref: ref}
}

func (s *stageMetadata) Ref() string {
	return s.ref
}

func (s *stageMetadata) Name() string {
	return s.name
}

func (s *stageMetadata) Category() string {
	return s.category
}

func (s *stageMetadata) SetName(name string) StageMetadata {
	s.name = name
	return s
}

func (s *stageMetadata) SetCategory(category string) StageMetadata {
	s.category = category
	return s
}
