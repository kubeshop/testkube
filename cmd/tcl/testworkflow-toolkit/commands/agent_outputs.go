// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//     https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package commands

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/data"
	"github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/agentoutputs"
)

// agentOutputsBlock is the language of the fenced block that holds the outputs in the answer.
const agentOutputsBlock = "testkube-outputs"

// agentOutputsFence finds each outputs block in an answer.
var agentOutputsFence = regexp.MustCompile("(?s)```" + agentOutputsBlock + "[ \t]*\r?\n(.*?)```")

// checkAgentOutputSchemas returns an error for a declaration that the step cannot keep. It runs
// before the turn starts, because a turn costs time and tokens. The processor makes the same
// check, but not for a default, which can be an expression until the pod resolves it.
func checkAgentOutputSchemas(schemas map[string]testworkflowsv1.ParameterSchema) error {
	if err := agentoutputs.CheckSchemas(schemas); err != nil {
		return err
	}
	for _, name := range agentoutputs.Names(schemas) {
		schema := schemas[name]
		if schema.Default != nil {
			if _, err := checkAgentOutputValue(schema, schema.Default.String()); err != nil {
				return fmt.Errorf("the default of the output %q is not valid: %w", name, err)
			}
		}
	}
	return nil
}

// agentOutputsContract is the text that the step adds to the prompt. It tells the AI Agent how
// to give the declared outputs.
func agentOutputsContract(schemas map[string]testworkflowsv1.ParameterSchema) string {
	if len(schemas) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n\nEnd your answer with one fenced code block that has the language `" + agentOutputsBlock + "`. ")
	b.WriteString("The block holds one JSON object with these fields, and nothing else:\n")
	for _, name := range agentoutputs.Names(schemas) {
		schema := schemas[name]
		typ := string(schema.Type)
		if typ == "" {
			typ = string(testworkflowsv1.ParameterTypeString)
		}
		fmt.Fprintf(&b, "- %q: %s", name, typ)
		if len(schema.Enum) > 0 {
			fmt.Fprintf(&b, ", one of %s", strings.Join(schema.Enum, ", "))
		}
		if schema.Pattern != "" {
			fmt.Fprintf(&b, ", matches the regular expression %s", schema.Pattern)
		}
		if schema.Default != nil {
			b.WriteString(", optional")
		}
		if schema.Description != "" {
			fmt.Fprintf(&b, ". %s", schema.Description)
		}
		b.WriteString("\n")
	}
	return b.String()
}

// parseAgentOutputs reads the fields of the outputs block that ends the answer. The answer can
// quote text from a tool, such as a test log, so a block with text after it does not count.
func parseAgentOutputs(answer string) (map[string]json.RawMessage, error) {
	blocks := agentOutputsFence.FindAllStringSubmatchIndex(answer, -1)
	if len(blocks) == 0 {
		return nil, fmt.Errorf("the answer has no `%s` block", agentOutputsBlock)
	}
	last := blocks[len(blocks)-1]
	if strings.TrimSpace(answer[last[1]:]) != "" {
		return nil, fmt.Errorf("the answer has text after its last `%s` block", agentOutputsBlock)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(answer[last[2]:last[3]]), &fields); err != nil {
		return nil, fmt.Errorf("the `%s` block is not one JSON object: %w", agentOutputsBlock, err)
	}
	return fields, nil
}

// agentOutputText returns the text of a JSON value. An output is text, so a string, a number
// and a boolean have one. An object, a list and null have none.
func agentOutputText(raw json.RawMessage) (string, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return "", err
	}
	switch v := value.(type) {
	case string:
		return v, nil
	case json.Number:
		return v.String(), nil
	case bool:
		return strconv.FormatBool(v), nil
	case nil:
		return "", errors.New("the value is null")
	default:
		return "", errors.New("the value is an object or a list. Give text, a number or a boolean")
	}
}

// checkAgentOutputValue checks the text of one output against its schema. It returns the value
// that the step writes.
func checkAgentOutputValue(schema testworkflowsv1.ParameterSchema, text string) (string, error) {
	// The init process trims an output, and the text can come from a tool.
	text = strings.TrimSpace(plainText(text))
	if len(text) > data.MaxOutputSize {
		return "", fmt.Errorf("the value has %d bytes, and a step output holds %d", len(text), data.MaxOutputSize)
	}
	if len(schema.Enum) > 0 && !slices.Contains(schema.Enum, text) {
		return "", fmt.Errorf("the value %q is not one of %s", text, strings.Join(schema.Enum, ", "))
	}

	switch schema.Type {
	case testworkflowsv1.ParameterTypeBoolean:
		if text != "true" && text != "false" {
			return "", fmt.Errorf("the value %q is not true or false", text)
		}
	case testworkflowsv1.ParameterTypeInteger:
		number, err := strconv.ParseInt(text, 10, 64)
		if err != nil {
			return "", fmt.Errorf("the value %q is not an integer", text)
		}
		if err := checkAgentOutputNumber(schema.ParameterNumberSchema, float64(number)); err != nil {
			return "", err
		}
	case testworkflowsv1.ParameterTypeNumber:
		number, err := strconv.ParseFloat(text, 64)
		if err != nil || math.IsNaN(number) || math.IsInf(number, 0) {
			return "", fmt.Errorf("the value %q is not a number", text)
		}
		if err := checkAgentOutputNumber(schema.ParameterNumberSchema, number); err != nil {
			return "", err
		}
	default:
		if err := checkAgentOutputString(schema.ParameterStringSchema, text); err != nil {
			return "", err
		}
	}
	return text, nil
}

func checkAgentOutputString(schema testworkflowsv1.ParameterStringSchema, text string) error {
	length := int64(utf8.RuneCountInString(text))
	if schema.MinLength != nil && length < *schema.MinLength {
		return fmt.Errorf("the value has %d characters, and the minimum is %d", length, *schema.MinLength)
	}
	if schema.MaxLength != nil && length > *schema.MaxLength {
		return fmt.Errorf("the value has %d characters, and the maximum is %d", length, *schema.MaxLength)
	}
	if schema.Pattern != "" {
		pattern, err := regexp.Compile(schema.Pattern)
		if err != nil {
			return fmt.Errorf("the pattern is not a regular expression: %w", err)
		}
		if !pattern.MatchString(text) {
			return fmt.Errorf("the value %q does not match the pattern %s", text, schema.Pattern)
		}
	}
	return nil
}

func checkAgentOutputNumber(schema testworkflowsv1.ParameterNumberSchema, number float64) error {
	if schema.Minimum != nil && number < float64(*schema.Minimum) {
		return fmt.Errorf("the value %v is below the minimum %d", number, *schema.Minimum)
	}
	if schema.Maximum != nil && number > float64(*schema.Maximum) {
		return fmt.Errorf("the value %v is above the maximum %d", number, *schema.Maximum)
	}
	if schema.ExclusiveMinimum != nil && number <= float64(*schema.ExclusiveMinimum) {
		return fmt.Errorf("the value %v is not above %d", number, *schema.ExclusiveMinimum)
	}
	if schema.ExclusiveMaximum != nil && number >= float64(*schema.ExclusiveMaximum) {
		return fmt.Errorf("the value %v is not below %d", number, *schema.ExclusiveMaximum)
	}
	if schema.MultipleOf != nil && *schema.MultipleOf != 0 && math.Mod(number, float64(*schema.MultipleOf)) != 0 {
		return fmt.Errorf("the value %v is not a multiple of %d", number, *schema.MultipleOf)
	}
	return nil
}

// agentOutputFields returns the fields that the AI Agent gave. The fields from its tool come
// first, because they are apart from the text. An older control plane gives none, and then the
// fields come from the block in the answer.
func agentOutputFields(fromTool, answer string) (map[string]json.RawMessage, error) {
	if strings.TrimSpace(fromTool) == "" {
		return parseAgentOutputs(answer)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(fromTool), &fields); err != nil {
		return nil, fmt.Errorf("the outputs from the tool of the AI Agent are not one JSON object: %w", err)
	}
	return fields, nil
}

// agentOutputsSchemaJSON encodes the declared outputs for the control plane.
func agentOutputsSchemaJSON(schemas map[string]testworkflowsv1.ParameterSchema) (string, error) {
	if len(schemas) == 0 {
		return "", nil
	}
	encoded, err := json.Marshal(schemas)
	if err != nil {
		return "", fmt.Errorf("cannot encode the outputs of the step: %w", err)
	}
	return string(encoded), nil
}

// resolveAgentOutputs returns the value of each declared output from the fields. A field that
// the answer does not have takes its default. The error names each output that has no valid
// value, so the author corrects the prompt or the schema in one pass.
func resolveAgentOutputs(schemas map[string]testworkflowsv1.ParameterSchema, fields map[string]json.RawMessage) (map[string]string, error) {
	values := make(map[string]string, len(schemas))
	var problems []string
	for _, name := range agentoutputs.Names(schemas) {
		schema := schemas[name]
		raw, ok := fields[name]
		if !ok {
			if schema.Default == nil {
				problems = append(problems, fmt.Sprintf("%q: the answer has no value", name))
				continue
			}
			raw, _ = json.Marshal(schema.Default.String())
		}
		text, err := agentOutputText(raw)
		if err == nil {
			text, err = checkAgentOutputValue(schema, text)
		}
		if err != nil {
			problems = append(problems, fmt.Sprintf("%q: %s", name, err.Error()))
			continue
		}
		values[name] = text
	}
	if len(problems) > 0 {
		return nil, errors.New(strings.Join(problems, "; "))
	}
	return values, nil
}

// agentOutputDefaults returns the default of each declared output that has one. The step writes
// them when the turn gives no valid outputs. A condition on an output that does not exist
// aborts the execution, and with a default a later step gets a known value, such as "no-go".
func agentOutputDefaults(schemas map[string]testworkflowsv1.ParameterSchema) map[string]string {
	values := make(map[string]string)
	for name, schema := range schemas {
		if schema.Default == nil {
			continue
		}
		if text, err := checkAgentOutputValue(schema, schema.Default.String()); err == nil {
			values[name] = text
		}
	}
	return values
}
