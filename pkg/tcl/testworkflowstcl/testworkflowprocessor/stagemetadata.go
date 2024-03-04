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

	SetName(name string, fallbacks ...string) StageMetadata
}

type stageMetadata struct {
	ref  string
	name string `expr:"template"`
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

func (s *stageMetadata) SetName(name string, fallbacks ...string) StageMetadata {
	s.name = name
	for i := 0; s.name == "" && i < len(fallbacks); i++ {
		s.name = fallbacks[i]
	}
	return s
}
