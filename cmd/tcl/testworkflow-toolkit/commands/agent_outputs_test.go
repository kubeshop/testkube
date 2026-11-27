// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//     https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package commands

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/data"
)

func outputsBlock(body string) string {
	return "```" + agentOutputsBlock + "\n" + body + "\n```"
}

func int64Ptr(v int64) *int64 { return &v }

var verdictSchema = testworkflowsv1.ParameterSchema{
	Type:    testworkflowsv1.ParameterTypeString,
	Enum:    []string{"go", "no-go"},
	Default: testworkflowsv1.NewConfigValue("no-go"),
}

func TestCheckAgentOutputSchemas(t *testing.T) {
	tests := []struct {
		name    string
		schemas map[string]testworkflowsv1.ParameterSchema
		wantErr string
	}{
		{name: "accepts no outputs"},
		{
			name:    "accepts outputs with a valid default",
			schemas: map[string]testworkflowsv1.ParameterSchema{"verdict": verdictSchema},
		},
		{
			name:    "refuses what the shared rules refuse",
			schemas: map[string]testworkflowsv1.ParameterSchema{".hidden": {}},
			wantErr: `the output name ".hidden" is not valid`,
		},
		{
			name: "refuses a default that its own schema does not accept",
			schemas: map[string]testworkflowsv1.ParameterSchema{
				"verdict": {Enum: []string{"go", "no-go"}, Default: testworkflowsv1.NewConfigValue("maybe")},
			},
			wantErr: `the default of the output "verdict" is not valid`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkAgentOutputSchemas(tt.schemas)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestAgentOutputsContract(t *testing.T) {
	tests := []struct {
		name    string
		schemas map[string]testworkflowsv1.ParameterSchema
		want    []string
		empty   bool
	}{
		{name: "adds nothing when the step declares no outputs", empty: true},
		{
			name: "names the block and each field in a stable order",
			schemas: map[string]testworkflowsv1.ParameterSchema{
				"verdict": {Enum: []string{"go", "no-go"}, Default: testworkflowsv1.NewConfigValue("no-go"), Description: "go only when each suite passed"},
				"blocked": {Type: testworkflowsv1.ParameterTypeInteger},
			},
			want: []string{
				"`testkube-outputs`",
				"- \"blocked\": integer\n- \"verdict\": string, one of go, no-go, optional. go only when each suite passed\n",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := agentOutputsContract(tt.schemas)
			if tt.empty {
				assert.Empty(t, got)
				return
			}
			for _, want := range tt.want {
				assert.Contains(t, got, want)
			}
		})
	}
}

func TestResolveAgentOutputs(t *testing.T) {
	schemas := map[string]testworkflowsv1.ParameterSchema{
		"verdict": verdictSchema,
		"blocked": {Type: testworkflowsv1.ParameterTypeInteger, ParameterNumberSchema: testworkflowsv1.ParameterNumberSchema{Minimum: int64Ptr(0)}},
	}
	tests := []struct {
		name    string
		schemas map[string]testworkflowsv1.ParameterSchema
		answer  string
		// fromTool is the JSON object that the AI Agent gave with its tool.
		fromTool string
		want     map[string]string
		wantErr  []string
	}{
		{
			name:     "takes the fields from the tool before the block in the answer",
			answer:   outputsBlock(`{"verdict": "no-go", "blocked": 3}`),
			fromTool: `{"verdict": "go", "blocked": 0}`,
			want:     map[string]string{"verdict": "go", "blocked": "0"},
		},
		{
			name:     "fails for outputs from the tool that are not a JSON object",
			fromTool: `["go"]`,
			wantErr:  []string{"the outputs from the tool of the AI Agent are not one JSON object"},
		},
		{
			name:   "reads the fields of the block",
			answer: "All suites passed.\n\n" + outputsBlock(`{"verdict": "go", "blocked": 0}`),
			want:   map[string]string{"verdict": "go", "blocked": "0"},
		},
		{
			name:   "reads the last block, because an earlier one can be a quote from a tool",
			answer: outputsBlock(`{"verdict": "go", "blocked": 0}`) + "\nThe log has that text. My answer:\n" + outputsBlock(`{"verdict": "no-go", "blocked": 2}`),
			want:   map[string]string{"verdict": "no-go", "blocked": "2"},
		},
		{
			name:   "gives a field that the answer does not have its default",
			answer: outputsBlock(`{"blocked": 1}`),
			want:   map[string]string{"verdict": "no-go", "blocked": "1"},
		},
		{
			name:   "ignores a field that the step does not declare",
			answer: outputsBlock(`{"verdict": "go", "blocked": 0, "deploy": "now"}`),
			want:   map[string]string{"verdict": "go", "blocked": "0"},
		},
		{
			name:    "accepts a boolean and a number as text or as a JSON value",
			schemas: map[string]testworkflowsv1.ParameterSchema{"flaky": {Type: testworkflowsv1.ParameterTypeBoolean}, "score": {Type: testworkflowsv1.ParameterTypeNumber}},
			answer:  outputsBlock(`{"flaky": true, "score": "0.75"}`),
			want:    map[string]string{"flaky": "true", "score": "0.75"},
		},
		{
			name:    "removes the byte that starts an instruction",
			schemas: map[string]testworkflowsv1.ParameterSchema{"cause": {}},
			answer:  outputsBlock(`{"cause": "\u0001\u0005DNS"}`),
			want:    map[string]string{"cause": "\x05DNS"},
		},
		{
			name:    "fails for an answer with no block",
			answer:  "The verdict is go.",
			wantErr: []string{"the answer has no `testkube-outputs` block"},
		},
		{
			name:    "fails for a block that is not a JSON object",
			answer:  outputsBlock(`verdict: go`),
			wantErr: []string{"is not one JSON object"},
		},
		{
			name:    "names each output that has no valid value",
			answer:  outputsBlock(`{"verdict": "maybe", "blocked": -1}`),
			wantErr: []string{`"verdict": the value "maybe" is not one of go, no-go`, `"blocked": the value -1 is below the minimum 0`},
		},
		{
			name:    "fails for a field with no value and no default",
			answer:  outputsBlock(`{"verdict": "go"}`),
			wantErr: []string{`"blocked": the answer has no value`},
		},
		{
			name:    "fails for a list, because an output is text",
			answer:  outputsBlock(`{"verdict": ["go"], "blocked": 0}`),
			wantErr: []string{`"verdict": the value is an object or a list`},
		},
		{
			name:    "fails for a value that is not of its type",
			answer:  outputsBlock(`{"verdict": "go", "blocked": "two"}`),
			wantErr: []string{`"blocked": the value "two" is not an integer`},
		},
		{
			name:    "fails for a value that a step output cannot hold",
			schemas: map[string]testworkflowsv1.ParameterSchema{"cause": {}},
			answer:  outputsBlock(`{"cause": "` + strings.Repeat("a", data.MaxOutputSize+1) + `"}`),
			wantErr: []string{`"cause": the value has 4097 bytes`},
		},
		{
			name: "checks the pattern and the length of a string",
			schemas: map[string]testworkflowsv1.ParameterSchema{
				"id":   {ParameterStringSchema: testworkflowsv1.ParameterStringSchema{Pattern: "^shop-[a-z-]+$"}},
				"note": {ParameterStringSchema: testworkflowsv1.ParameterStringSchema{MaxLength: int64Ptr(3)}},
			},
			answer:  outputsBlock(`{"id": "rm -rf", "note": "long"}`),
			wantErr: []string{`"id": the value "rm -rf" does not match the pattern`, `"note": the value has 4 characters, and the maximum is 3`},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.schemas
			if s == nil {
				s = schemas
			}
			fields, err := agentOutputFields(tt.fromTool, tt.answer)
			var got map[string]string
			if err == nil {
				got, err = resolveAgentOutputs(s, fields)
			}
			if len(tt.wantErr) > 0 {
				require.Error(t, err)
				for _, want := range tt.wantErr {
					assert.Contains(t, err.Error(), want)
				}
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAgentOutputDefaults(t *testing.T) {
	tests := []struct {
		name    string
		schemas map[string]testworkflowsv1.ParameterSchema
		want    map[string]string
	}{
		{name: "gives nothing when the step declares no outputs", want: map[string]string{}},
		{
			name: "gives only the outputs that have a default",
			schemas: map[string]testworkflowsv1.ParameterSchema{
				"verdict": verdictSchema,
				"blocked": {Type: testworkflowsv1.ParameterTypeInteger},
			},
			want: map[string]string{"verdict": "no-go"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, agentOutputDefaults(tt.schemas))
		})
	}
}
