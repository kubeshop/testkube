// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package data

import (
	"encoding/json"
	"fmt"
)

// TODO: Replace prefix with something less common
const (
	InstructionPrefix = ";;"
	HintPrefix        = ";"
)

func EmitOutput(ref string, name string, value interface{}) {
	j, err := json.Marshal(value)
	if err != nil {
		panic(fmt.Sprintf("error while marshalling reference: %v", err))
	}
	fmt.Printf("\n%s%s;%s:%s;\n", InstructionPrefix, ref, name, string(j))
}

func EmitHint(ref string, name string) {
	fmt.Printf("\n%s%s%s;%s;\n", InstructionPrefix, HintPrefix, ref, name)
}

func EmitHintDetails(ref string, name string, value interface{}) {
	j, err := json.Marshal(value)
	if err != nil {
		panic(fmt.Sprintf("error while marshalling reference: %v", err))
	}
	fmt.Printf("\n%s%s%s;%s:%s;\n", InstructionPrefix, HintPrefix, ref, name, string(j))
}
