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

	"github.com/kballard/go-shellquote"
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
		return NewValue(str), nil
	},
	"int": func(value ...StaticValue) (Expression, error) {
		if len(value) != 1 {
			return nil, fmt.Errorf(`"int" function expects 1 argument, %d provided`, len(value))
		}
		v, err := value[0].IntValue()
		if err != nil {
			return nil, err
		}
		return NewValue(v), nil
	},
	"float": func(value ...StaticValue) (Expression, error) {
		if len(value) != 1 {
			return nil, fmt.Errorf(`"float" function expects 1 argument, %d provided`, len(value))
		}
		v, err := value[0].FloatValue()
		if err != nil {
			return nil, err
		}
		return NewValue(v), nil
	},
	"tojson": func(value ...StaticValue) (Expression, error) {
		if len(value) != 1 {
			return nil, fmt.Errorf(`"tojson" function expects 1 argument, %d provided`, len(value))
		}
		b, err := json.Marshal(value[0].Value())
		if err != nil {
			return nil, fmt.Errorf(`"tojson" function had problem unmarshalling: %s`, err.Error())
		}
		return NewValue(string(b)), nil
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
		return NewValue(v), nil
	},
	"shellquote": func(value ...StaticValue) (Expression, error) {
		args := make([]string, len(value))
		for i := range value {
			args[i], _ = value[i].StringValue()
		}
		return NewValue(shellquote.Join(args...)), nil
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
