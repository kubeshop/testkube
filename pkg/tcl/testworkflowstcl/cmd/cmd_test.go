// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package cmd

import (
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCmd_GetJWTPayload(t *testing.T) {

	t.Run("valid jwt token", func(t *testing.T) {

		header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
		payload := base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"1234567890","name":"John Doe","email":"fake@email.com"}`))
		signature := "signature"

		token := fmt.Sprintf("%s.%s.%s", header, payload, signature)

		expectedPayload := map[string]interface{}{
			"sub":   "1234567890",
			"name":  "John Doe",
			"email": "fake@email.com",
		}

		result, err := getJWTPayload(token)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		assert.Equal(t, expectedPayload, result)
	})

	t.Run("invalid jwt token", func(t *testing.T) {

		token := "invalid.token"

		_, err := getJWTPayload(token)
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		expectedError := "invalid token format"
		assert.EqualError(t, err, expectedError)
	})

	t.Run("invalid base64 payload", func(t *testing.T) {

		header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
		payload := "invalidbase64"
		signature := "signature"

		token := fmt.Sprintf("%s.%s.%s", header, payload, signature)

		_, err := getJWTPayload(token)
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		expectedError := "failed to decode payload"
		assert.ErrorContains(t, err, expectedError)
	})

	t.Run("invalid json", func(t *testing.T) {

		header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
		payload := base64.RawURLEncoding.EncodeToString([]byte(`invalid-json`))
		signature := "signature"

		token := fmt.Sprintf("%s.%s.%s", header, payload, signature)

		_, err := getJWTPayload(token)
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		expectedError := "failed to parse payload JSON"
		assert.ErrorContains(t, err, expectedError)
	})
}
