// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package testworkflowprocessor

type Signature interface {
	Ref() string
	Name() string
	Optional() bool
	Negative() bool
	Children() []Signature
}

type signature struct {
	RefValue      string      `json:"ref"`
	NameValue     string      `json:"name,omitempty"`
	OptionalValue bool        `json:"optional,omitempty"`
	NegativeValue bool        `json:"negative,omitempty"`
	ChildrenValue []Signature `json:"children,omitempty"`
}

func (s *signature) Ref() string {
	return s.RefValue
}

func (s *signature) Name() string {
	return s.NameValue
}

func (s *signature) Optional() bool {
	return s.OptionalValue
}

func (s *signature) Negative() bool {
	return s.NegativeValue
}

func (s *signature) Children() []Signature {
	return s.ChildrenValue
}
