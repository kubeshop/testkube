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
	"strings"
)

const (
	InstructionPrefix         = "\u0001\u0005"
	HintPrefix                = "\u0006"
	InstructionSeparator      = "\u0003"
	InstructionValueSeparator = "\u0004"
)

func SprintOutput(ref string, name string, value interface{}) string {
	j, err := json.Marshal(value)
	if err != nil {
		panic(fmt.Sprintf("error while marshalling reference: %v", err))
	}
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(InstructionPrefix)
	sb.WriteString(ref)
	sb.WriteString(InstructionSeparator)
	sb.WriteString(name)
	sb.WriteString(InstructionValueSeparator)
	sb.Write(j)
	sb.WriteString(InstructionSeparator)
	sb.WriteString("\n")
	return sb.String()
}

func SprintHint(ref string, name string) string {
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(InstructionPrefix)
	sb.WriteString(HintPrefix)
	sb.WriteString(ref)
	sb.WriteString(InstructionSeparator)
	sb.WriteString(name)
	sb.WriteString(InstructionSeparator)
	sb.WriteString("\n")
	return sb.String()
}

func SprintHintDetails(ref string, name string, value interface{}) string {
	j, err := json.Marshal(value)
	if err != nil {
		panic(fmt.Sprintf("error while marshalling reference: %v", err))
	}
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(InstructionPrefix)
	sb.WriteString(HintPrefix)
	sb.WriteString(ref)
	sb.WriteString(InstructionSeparator)
	sb.WriteString(name)
	sb.WriteString(InstructionValueSeparator)
	sb.Write(j)
	sb.WriteString(InstructionSeparator)
	sb.WriteString("\n")
	return sb.String()
}

func PrintOutput(ref string, name string, value interface{}) {
	fmt.Print(SprintOutput(ref, name, value))
}

func PrintHint(ref string, name string) {
	fmt.Print(SprintHint(ref, name))
}

func PrintHintDetails(ref string, name string, value interface{}) {
	fmt.Print(SprintHintDetails(ref, name, value))
}
