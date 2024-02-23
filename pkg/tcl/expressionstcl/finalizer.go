// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//     https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package expressionstcl

import (
	"fmt"
)

type finalizer struct {
	machine MachineCore
}

func (f *finalizer) Get(name string) (Expression, bool, error) {
	v, ok, err := f.machine.Get(name)
	if !ok && err == nil {
		return NewNone(), true, nil
	}
	return v, ok, err
}

func (f *finalizer) Call(name string, args ...StaticValue) (Expression, bool, error) {
	v, ok, err := f.machine.Call(name, args...)
	if !ok && err == nil {
		return nil, true, fmt.Errorf(`"%s" function not resolved`, name)
	}
	return v, ok, err
}
