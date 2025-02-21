package stage

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
	Sequence() []Signature
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

func (s *signature) Sequence() []Signature {
	result := []Signature{s}
	for i := range s.ChildrenValue {
		result = append(result, s.ChildrenValue[i].Sequence()...)
	}
	return result
}

func MapSignatureToSequence(v []Signature) []Signature {
	if len(v) == 0 {
		return nil
	}
	return (&signature{ChildrenValue: v}).Sequence()[1:]
}

func MapSignatureListToInternal(v []Signature) []testkube.TestWorkflowSignature {
	r := make([]testkube.TestWorkflowSignature, len(v))
	for i := range v {
		r[i] = v[i].ToInternal()
	}
	return r
}

func MapSignatureList(v []testkube.TestWorkflowSignature) []Signature {
	r := make([]Signature, len(v))
	for i := range v {
		r[i] = Signature(&signature{
			RefValue:      v[i].Ref,
			NameValue:     v[i].Name,
			CategoryValue: v[i].Category,
			OptionalValue: v[i].Optional,
			NegativeValue: v[i].Negative,
			ChildrenValue: MapSignatureList(v[i].Children),
		})
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
