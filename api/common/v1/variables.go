package v1

import (
	corev1 "k8s.io/api/core/v1"
)

// +kubebuilder:object:generate=true
type Variable struct {
	// variable type
	Type_ string `json:"type,omitempty"`
	// variable name
	Name string `json:"name,omitempty"`
	// variable string value
	Value string `json:"value,omitempty"`
	// or load it from var source
	ValueFrom corev1.EnvVarSource `json:"valueFrom,omitempty"`
}

const (
	VariableTypeBasic  = "basic"
	VariableTypeSecret = "secret"
)
