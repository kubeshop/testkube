// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package data

import (
	"fmt"
	"os"
	"strconv"
	"strings"
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
		fmt.Printf("invalid '%s' provided: '%s': %v\n", key, str, err)
		os.Exit(155)
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
