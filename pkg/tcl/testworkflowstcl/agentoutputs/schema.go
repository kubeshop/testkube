// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//     https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package agentoutputs

import (
	"fmt"
	"regexp"
	"sort"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
)

// name accepts the names that the init process reads as a step output. An output is a file, and
// the init process skips a file whose name starts with a dot.
var name = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_-]*$`)

// Names returns the names of the declared outputs in a stable order.
func Names(schemas map[string]testworkflowsv1.ParameterSchema) []string {
	names := make([]string, 0, len(schemas))
	for n := range schemas {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// CheckSchemas returns an error for a declaration that an agent step cannot keep.
func CheckSchemas(schemas map[string]testworkflowsv1.ParameterSchema) error {
	for _, n := range Names(schemas) {
		schema := schemas[n]
		if !name.MatchString(n) {
			return fmt.Errorf("the output name %q is not valid: use letters, digits, '_' and '-', and start with a letter, a digit or '_'", n)
		}
		switch schema.Type {
		case "", testworkflowsv1.ParameterTypeString, testworkflowsv1.ParameterTypeInteger,
			testworkflowsv1.ParameterTypeNumber, testworkflowsv1.ParameterTypeBoolean:
		default:
			return fmt.Errorf("the output %q has the type %q: use string, integer, number or boolean", n, schema.Type)
		}
		if schema.Pattern != "" {
			if _, err := regexp.Compile(schema.Pattern); err != nil {
				return fmt.Errorf("the output %q has a pattern that is not a regular expression: %w", n, err)
			}
		}
	}
	return nil
}
