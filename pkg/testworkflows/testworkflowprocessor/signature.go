// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package testworkflowprocessor

import (
	"encoding/json"
	"maps"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

type Signature interface {
	Ref() string
	Name() string
	Category() string
	Optional() bool
	Negative() bool
	Children() []Signature
	ToInternal() testkube.TestWorkflowSignature
}

type signature struct {
	RefValue      string      `json:"ref"`
	NameValue     string      `json:"name,omitempty"`
	CategoryValue string      `json:"category,omitempty"`
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

func (s *signature) Category() string {
	return s.CategoryValue
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

func (s *signature) ToInternal() testkube.TestWorkflowSignature {
	return testkube.TestWorkflowSignature{
		Ref:      s.RefValue,
		Name:     s.NameValue,
		Category: s.CategoryValue,
		Optional: s.OptionalValue,
		Negative: s.NegativeValue,
		Children: MapSignatureListToInternal(s.ChildrenValue),
	}
}

func MapSignatureListToInternal(v []Signature) []testkube.TestWorkflowSignature {
	r := make([]testkube.TestWorkflowSignature, len(v))
	for i := range v {
		r[i] = v[i].ToInternal()
	}
	return r
}

func MapSignatureListToStepResults(v []Signature) map[string]testkube.TestWorkflowStepResult {
	r := map[string]testkube.TestWorkflowStepResult{}
	for _, s := range v {
		r[s.Ref()] = testkube.TestWorkflowStepResult{
			Status: common.Ptr(testkube.QUEUED_TestWorkflowStepStatus),
		}
		maps.Copy(r, MapSignatureListToStepResults(s.Children()))
	}
	return r
}

type rawSignature struct {
	RefValue      string         `json:"ref"`
	NameValue     string         `json:"name,omitempty"`
	CategoryValue string         `json:"category,omitempty"`
	OptionalValue bool           `json:"optional,omitempty"`
	NegativeValue bool           `json:"negative,omitempty"`
	ChildrenValue []rawSignature `json:"children,omitempty"`
}

func rawSignatureToSignature(sig rawSignature) Signature {
	ch := make([]Signature, len(sig.ChildrenValue))
	for i, v := range sig.ChildrenValue {
		ch[i] = rawSignatureToSignature(v)
	}
	return &signature{
		RefValue:      sig.RefValue,
		NameValue:     sig.NameValue,
		CategoryValue: sig.CategoryValue,
		OptionalValue: sig.OptionalValue,
		NegativeValue: sig.NegativeValue,
		ChildrenValue: ch,
	}
}

func GetSignatureFromJSON(v []byte) ([]Signature, error) {
	var sig []rawSignature
	err := json.Unmarshal(v, &sig)
	if err != nil {
		return nil, err
	}
	res := make([]Signature, len(sig))
	for i := range sig {
		res[i] = rawSignatureToSignature(sig[i])
	}
	return res, err
}

func GetVirtualSignature(children []Signature) Signature {
	return &signature{ChildrenValue: children}
}
