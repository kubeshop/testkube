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
	SafeResolve(...MachineCore) (Expression, bool, error)
	Resolve(...MachineCore) (Expression, error)
	Static() StaticValue
	Accessors() map[string]struct{}
	Functions() map[string]struct{}
}

type StringAwareExpression interface {
	WillBeString() bool
}

//go:generate mockgen -destination=./mock_staticvalue.go -package=expressionstcl "github.com/kubeshop/testkube/pkg/tcl/expressionstcl" StaticValue
type StaticValue interface {
	Expression
	StringAwareExpression
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
