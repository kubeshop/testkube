// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package expressionstcl

import (
	"encoding/base64"
	"encoding/json"

	"github.com/pkg/errors"
)

// EncodeBase64JSON encodes data as base64-encoded JSON.
// This is used to prevent testworkflow-init from prematurely resolving expressions
// in command arguments. The encoded data is passed through init unchanged, then
// decoded in the toolkit command where the proper expression context is available.
func EncodeBase64JSON(data interface{}) (string, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", errors.Wrap(err, "marshaling to JSON")
	}

	return base64.StdEncoding.EncodeToString(jsonData), nil
}

// DecodeBase64JSON decodes base64 encoded JSON data into the target interface.
// Used by toolkit commands to decode arguments that were hidden from testworkflow-init's
// expression resolver.
func DecodeBase64JSON(encoded string, target interface{}) error {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return errors.Wrapf(err, "decoding base64 (length=%d)", len(encoded))
	}

	if err := json.Unmarshal(decoded, target); err != nil {
		return errors.Wrap(err, "parsing JSON after base64 decode")
	}

	return nil
}
