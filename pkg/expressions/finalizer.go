package expressions

import (
	"errors"
)

type finalizer struct {
	handler FinalizerFn
}

type finalizerItem struct {
	function bool
	name     string
}

type FinalizerItem interface {
	Name() string
	IsFunction() bool
}

type FinalizerResult int8

const (
	FinalizerResultFail     FinalizerResult = -1
	FinalizerResultNone     FinalizerResult = 0
	FinalizerResultPreserve FinalizerResult = 1
)

type FinalizerFn = func(item FinalizerItem) FinalizerResult

func NewFinalizer(fn FinalizerFn) Machine {
	return &finalizer{handler: fn}
}

func (f *finalizer) Get(name string) (Expression, bool, error) {
	result := f.handler(finalizerItem{name: name})
	switch result {
	case FinalizerResultFail:
		return nil, true, errors.New("unknown variable")
	case FinalizerResultNone:
		return None, true, nil
	}
	return nil, false, nil
}

func (f *finalizer) Call(name string, _ []CallArgument) (Expression, bool, error) {
	result := f.handler(finalizerItem{function: true, name: name})
	switch result {
	case FinalizerResultFail:
		return nil, true, errors.New("unknown function")
	case FinalizerResultNone:
		return None, true, nil
	}
	return nil, false, nil
}

func (f finalizerItem) IsFunction() bool {
	return f.function
}

func (f finalizerItem) Name() string {
	return f.name
}

func FinalizerFailFn(_ FinalizerItem) FinalizerResult {
	return FinalizerResultFail
}

func FinalizerNoneFn(_ FinalizerItem) FinalizerResult {
	return FinalizerResultNone
}

var FinalizerFail = NewFinalizer(FinalizerFailFn)
var FinalizerNone = NewFinalizer(FinalizerNoneFn)
