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

func EmitReferenceFor(ref string, name string) {
	fmt.Printf(";;%s;%s\n", ref, name)
}

func EmitReferenceDetailsFor(ref string, name string, value interface{}) {
	j, err := json.Marshal(value)
	if err != nil {
		panic(fmt.Sprintf("error while marshalling reference: %v", err))
	}
	fmt.Printf(";;%s;%s;%s\n", ref, name, string(j))
}

func EmitReference(name string) {
	EmitReferenceFor(Step.Ref, name)
}

func EmitReferenceDetails(name string, value interface{}) {
	EmitReferenceDetailsFor(Step.Ref, name, value)
}
