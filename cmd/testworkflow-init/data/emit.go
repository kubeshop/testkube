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

var StartHintRe = regexp.MustCompile(fmt.Sprintf(`^%s%s([^%s]+)%sstart%s$`,
	InstructionPrefix, HintPrefix, InstructionSeparator, InstructionSeparator, InstructionSeparator))

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

func MayBeInstruction(line []byte) bool {
	if len(line) >= len(InstructionPrefix) {
		for i := 0; i < len(InstructionPrefix); i++ {
			if line[i] != InstructionPrefix[i] {
				return false
			}
		}
	}
	return true
}

func DetectInstruction(line []byte) (*Instruction, bool, error) {
	// Fast check to avoid regexes
	if len(line) < 4 || !MayBeInstruction(line) {
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
