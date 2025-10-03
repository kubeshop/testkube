package v1

import (
	"k8s.io/apimachinery/pkg/util/intstr"
)

// +kubebuilder:validation:Enum=string;integer;number;boolean
type ParameterType string

const (
	ParameterTypeString  ParameterType = "string"
	ParameterTypeInteger ParameterType = "integer"
	ParameterTypeNumber  ParameterType = "number"
	ParameterTypeBoolean ParameterType = "boolean"
)

type ParameterStringSchema struct {
	// predefined format for the string
	Format string `json:"format,omitempty"`
	// regular expression to match
	Pattern string `json:"pattern,omitempty"`
	// minimum length for the string
	MinLength *int64 `json:"minLength,omitempty"`
	// maximum length for the string
	MaxLength *int64 `json:"maxLength,omitempty"`
}

type ParameterNumberSchema struct {
	// minimum value for the number (inclusive)
	Minimum *int64 `json:"minimum,omitempty"`
	// maximum value for the number (inclusive)
	Maximum *int64 `json:"maximum,omitempty"`
	// minimum value for the number (exclusive)
	ExclusiveMinimum *int64 `json:"exclusiveMinimum,omitempty"`
	// maximum value for the number (exclusive)
	ExclusiveMaximum *int64 `json:"exclusiveMaximum,omitempty"`
	// the number needs to be multiple of this value
	MultipleOf *int64 `json:"multipleOf,omitempty"`
}

type ParameterSchema struct {
	// parameter description
	Description string `json:"description,omitempty"`
	// type of the parameter
	// +kubebuilder:default=string
	Type ParameterType `json:"type,omitempty"`
	// the list of allowed values, when limited
	Enum []string `json:"enum,omitempty"`
	// exemplary value
	Example *intstr.IntOrString `json:"example,omitempty"`
	// default value - if not provided, the parameter is required
	// +kubebuilder:validation:XIntOrString
	Default *intstr.IntOrString `json:"default,omitempty" expr:"template"`
	// whether this value should be stored in the secret
	Sensitive bool `json:"sensitive,omitempty"`

	ParameterStringSchema `json:",inline" expr:"include"`
	ParameterNumberSchema `json:",inline" expr:"include"`
}
