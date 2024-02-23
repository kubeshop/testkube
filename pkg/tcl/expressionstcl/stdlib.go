// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//     https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package expressionstcl

import (
	"encoding/json"
	"fmt"
)

type stdMachine struct{}

var StdLibMachine = &stdMachine{}

var stdFunctions = map[string]func(...StaticValue) (Expression, error){
	"string": func(value ...StaticValue) (Expression, error) {
		str := ""
		for i := range value {
			next, _ := value[i].StringValue()
			str += next
		}
		return newStatic(str), nil
	},
	"int": func(value ...StaticValue) (Expression, error) {
		if len(value) != 1 {
			return nil, fmt.Errorf(`"int" function expects 1 argument, %d provided`, len(value))
		}
		v, err := value[0].IntValue()
		if err != nil {
			return nil, err
		}
		return newStatic(v), nil
	},
	"float": func(value ...StaticValue) (Expression, error) {
		if len(value) != 1 {
			return nil, fmt.Errorf(`"float" function expects 1 argument, %d provided`, len(value))
		}
		v, err := value[0].FloatValue()
		if err != nil {
			return nil, err
		}
		return newStatic(v), nil
	},
	"tojson": func(value ...StaticValue) (Expression, error) {
		if len(value) != 1 {
			return nil, fmt.Errorf(`"tojson" function expects 1 argument, %d provided`, len(value))
		}
		b, err := json.Marshal(value[0].Value())
		if err != nil {
			return nil, fmt.Errorf(`"tojson" function had problem unmarshalling: %s`, err.Error())
		}
		return newStatic(string(b)), nil
	},
	"json": func(value ...StaticValue) (Expression, error) {
		if len(value) != 1 {
			return nil, fmt.Errorf(`"json" function expects 1 argument, %d provided`, len(value))
		}
		if !value[0].IsString() {
			return nil, fmt.Errorf(`"json" function argument should be a string`)
		}
		var v interface{}
		err := json.Unmarshal([]byte(value[0].Value().(string)), &v)
		if err != nil {
			return nil, fmt.Errorf(`"json" function had problem unmarshalling: %s`, err.Error())
		}
		return newStatic(v), nil
	},
}

func (*stdMachine) Get(name string) (Expression, bool, error) {
	return nil, false, nil
}

func (*stdMachine) Call(name string, args ...StaticValue) (Expression, bool, error) {
	fn, ok := stdFunctions[name]
	if ok {
		exp, err := fn(args...)
		return exp, true, err
	}
	return nil, false, nil
}
