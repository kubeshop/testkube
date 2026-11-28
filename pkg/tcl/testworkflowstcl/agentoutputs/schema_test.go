// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//     https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package agentoutputs

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
)

func TestNames(t *testing.T) {
	tests := []struct {
		name    string
		schemas map[string]testworkflowsv1.ParameterSchema
		want    []string
	}{
		{name: "gives no name for no outputs", want: []string{}},
		{name: "sorts the names", schemas: map[string]testworkflowsv1.ParameterSchema{"verdict": {}, "blocked": {}}, want: []string{"blocked", "verdict"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, Names(tt.schemas))
		})
	}
}

func TestCheckSchemas(t *testing.T) {
	tests := []struct {
		name    string
		schemas map[string]testworkflowsv1.ParameterSchema
		wantErr string
	}{
		{name: "accepts no outputs"},
		{
			name: "accepts each type, and a schema with no type",
			schemas: map[string]testworkflowsv1.ParameterSchema{
				"verdict": {Type: testworkflowsv1.ParameterTypeString, Enum: []string{"go", "no-go"}},
				"count":   {Type: testworkflowsv1.ParameterTypeInteger},
				"score":   {Type: testworkflowsv1.ParameterTypeNumber},
				"flaky":   {Type: testworkflowsv1.ParameterTypeBoolean},
				"cause_1": {},
			},
		},
		{
			name:    "refuses a name that the init process does not read",
			schemas: map[string]testworkflowsv1.ParameterSchema{".hidden": {}},
			wantErr: `the output name ".hidden" is not valid`,
		},
		{
			name:    "refuses a name with a path",
			schemas: map[string]testworkflowsv1.ParameterSchema{"../verdict": {}},
			wantErr: `the output name "../verdict" is not valid`,
		},
		{
			name:    "refuses a type that an output cannot have",
			schemas: map[string]testworkflowsv1.ParameterSchema{"list": {Type: "array"}},
			wantErr: `the output "list" has the type "array"`,
		},
		{
			name:    "refuses a pattern that does not compile",
			schemas: map[string]testworkflowsv1.ParameterSchema{"id": {ParameterStringSchema: testworkflowsv1.ParameterStringSchema{Pattern: "("}}},
			wantErr: `the output "id" has a pattern that is not a regular expression`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckSchemas(tt.schemas)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}
