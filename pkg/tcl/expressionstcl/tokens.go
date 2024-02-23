// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//     https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package expressionstcl

type tokenType uint8

const (
	// Primitives
	tokenTypeAccessor tokenType = iota
	tokenTypeJson

	// Math
	tokenTypeNot
	tokenTypeMath
	tokenTypeOpen
	tokenTypeClose

	// Logical
	tokenTypeTernary
	tokenTypeTernarySeparator

	// Functions
	tokenTypeComma
)

type token struct {
	Type  tokenType
	Value interface{}
}

var (
	tokenNot              = token{Type: tokenTypeNot}
	tokenOpen             = token{Type: tokenTypeOpen}
	tokenClose            = token{Type: tokenTypeClose}
	tokenTernary          = token{Type: tokenTypeTernary}
	tokenTernarySeparator = token{Type: tokenTypeTernarySeparator}
	tokenComma            = token{Type: tokenTypeComma}
)

func tokenMath(op string) token {
	return token{Type: tokenTypeMath, Value: op}
}

func tokenJson(value interface{}) token {
	return token{Type: tokenTypeJson, Value: value}
}

func tokenAccessor(value interface{}) token {
	return token{Type: tokenTypeAccessor, Value: value}
}
