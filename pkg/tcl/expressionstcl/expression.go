// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//     https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package expressionstcl

//go:generate mockgen -destination=./mock_expression.go -package=expressionstcl "github.com/kubeshop/testkube/pkg/tcl/expressionstcl" Expression
type Expression interface {
	String() string
	SafeString() string
	Template() string
	Type() Type
	SafeResolve(...Machine) (Expression, bool, error)
	Resolve(...Machine) (Expression, error)
	Static() StaticValue
	Accessors() map[string]struct{}
	Functions() map[string]struct{}
}

type Type string

const (
	TypeUnknown Type = ""
	TypeBool    Type = "bool"
	TypeString  Type = "string"
	TypeFloat64 Type = "float64"
	TypeInt64   Type = "int64"
)

//go:generate mockgen -destination=./mock_staticvalue.go -package=expressionstcl "github.com/kubeshop/testkube/pkg/tcl/expressionstcl" StaticValue
type StaticValue interface {
	Expression
	IsNone() bool
	IsString() bool
	IsBool() bool
	IsInt() bool
	IsNumber() bool
	IsMap() bool
	IsSlice() bool
	Value() interface{}
	BoolValue() (bool, error)
	IntValue() (int64, error)
	FloatValue() (float64, error)
	StringValue() (string, error)
	MapValue() (map[string]interface{}, error)
	SliceValue() ([]interface{}, error)
}
