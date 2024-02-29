// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package data

import (
	"strconv"
	"strings"

	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-init/output"
)

func getStr(config map[string]string, key string, defaultValue string) string {
	val, ok := config[key]
	if !ok {
		return defaultValue
	}
	return val
}

func getInt(config map[string]string, key string, defaultValue int) int {
	str := getStr(config, key, "")
	if str == "" {
		return defaultValue
	}
	val, err := strconv.Atoi(str)
	if err != nil {
		output.Failf(output.CodeInputError, "invalid '%s' provided: '%s': %v", key, str, err)
	}
	return val
}

func getBool(config map[string]string, key string, defaultValue bool) bool {
	str := getStr(config, key, "")
	if str == "" {
		return defaultValue
	}
	return strings.ToLower(str) == "true" || str == "1"
}
