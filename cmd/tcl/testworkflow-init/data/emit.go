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

func EmitOutput(ref string, name string, value interface{}) {
	j, err := json.Marshal(value)
	if err != nil {
		panic(fmt.Sprintf("error while marshalling reference: %v", err))
	}
	fmt.Printf(";;%s;%s:%s;", ref, name, string(j))
}

func EmitHint(ref string, name string) {
	fmt.Printf(";;;%s;%s;", ref, name)
}

func EmitHintDetails(ref string, name string, value interface{}) {
	j, err := json.Marshal(value)
	if err != nil {
		panic(fmt.Sprintf("error while marshalling reference: %v", err))
	}
	fmt.Printf(";;;%s;%s:%s;", ref, name, string(j))
}
