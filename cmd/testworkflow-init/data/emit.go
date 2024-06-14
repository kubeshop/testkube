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
	"regexp"
	"strings"
)

const (
	InstructionPrefix         = "\u0001\u0005"
	HintPrefix                = "\u0006"
	InstructionSeparator      = "\u0003"
	InstructionValueSeparator = "\u0004"
)

var instructionRe = regexp.MustCompile(fmt.Sprintf(`^%s(%s)?([^%s]+)%s([a-zA-Z0-9-_.]+)(?:%s([^\n]+))?%s$`,
	InstructionPrefix, HintPrefix, InstructionSeparator, InstructionSeparator, InstructionValueSeparator, InstructionSeparator))

type Instruction struct {
	Ref   string
	Name  string
	Value interface{}
}

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

func DetectInstruction(line []byte) (*Instruction, bool, error) {
	// Fast check to avoid regexes
	if len(line) < 4 || string(line[0:len(InstructionPrefix)]) != InstructionPrefix || string(line[:len(InstructionPrefix)]) != InstructionPrefix {
		return nil, false, nil
	}
	// Parse the line
	v := instructionRe.FindSubmatch(line)
	if v == nil {
		return nil, false, nil
	}
	isHint := string(v[1]) == HintPrefix
	instruction := &Instruction{
		Ref:  string(v[2]),
		Name: string(v[3]),
	}
	if len(v) > 4 && v[4] != nil {
		err := json.Unmarshal(v[4], &instruction.Value)
		if err != nil {
			return instruction, isHint, err
		}
	}
	return instruction, isHint, nil
}
