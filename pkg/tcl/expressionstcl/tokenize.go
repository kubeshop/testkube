// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//     https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package expressionstcl

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
)

var mathOperatorRe = regexp.MustCompile(`^(?:!=|<>|==|>=|<=|&&|\*\*|\|\||[+\-*/><=%])`)
var noneRe = regexp.MustCompile(`^null(?:[^a-zA-Z\d_.]|$)`)
var jsonValueRe = regexp.MustCompile(`^(?:["{\[\d]|((?:true|false)(?:[^a-zA-Z\d_.]|$)))`)
var accessorRe = regexp.MustCompile(`^[a-zA-Z\d_](?:[a-zA-Z\d_.]*[a-zA-Z\d_])?`)
var spaceRe = regexp.MustCompile(`^\s+`)

func tokenizeNext(exp string, i int) (token, int, error) {
	for i < len(exp) {
		switch true {
		case exp[i] == ',':
			return tokenComma, i + 1, nil
		case exp[i] == '(':
			return tokenOpen, i + 1, nil
		case exp[i] == ')':
			return tokenClose, i + 1, nil
		case exp[i] == ':':
			return tokenTernarySeparator, i + 1, nil
		case mathOperatorRe.MatchString(exp[i:]):
			op := mathOperatorRe.FindString(exp[i:])
			return tokenMath(op), i + len(op), nil
		case exp[i] == '?':
			return tokenTernary, i + 1, nil
		case exp[i] == '!':
			return tokenNot, i + 1, nil
		case spaceRe.MatchString(exp[i:]):
			space := spaceRe.FindString(exp[i:])
			i += len(space)
		case noneRe.MatchString(exp[i:]):
			return tokenJson(noneValue), i + 4, nil
		case jsonValueRe.MatchString(exp[i:]):
			// Allow multi-line string with literal \n
			// TODO: Optimize, and allow deeper in the tree
			appended := 0
			if exp[i] == '"' {
				inside := true
				for index := i + 1; inside && index < len(exp); index++ {
					if exp[index] == '\\' {
						index++
					} else if exp[index] == '"' {
						inside = false
					} else if exp[index] == '\n' {
						exp = exp[0:index] + "\\n" + exp[index+1:]
						appended++
					} else if exp[index] == '\t' {
						exp = exp[0:index] + "\\t" + exp[index+1:]
						appended++
					}
				}
			}
			decoder := json.NewDecoder(bytes.NewBuffer([]byte(exp[i:])))
			var val interface{}
			err := decoder.Decode(&val)
			if err != nil {
				return token{}, i, fmt.Errorf("error while decoding JSON from index %d in expression: %s: %s", i, exp, err.Error())
			}
			return tokenJson(val), i + int(decoder.InputOffset()) - appended, nil
		case accessorRe.MatchString(exp[i:]):
			acc := accessorRe.FindString(exp[i:])
			return tokenAccessor(acc), i + len(acc), nil
		default:
			return token{}, i, fmt.Errorf("unknown character at index %d in expression: %s", i, exp)
		}
	}
	return token{}, 0, io.EOF
}

func tokenize(exp string, index int) (tokens []token, i int, err error) {
	tokens = make([]token, 0)
	var t token
	for i = index; i < len(exp); {
		t, i, err = tokenizeNext(exp, i)
		if err != nil {
			if err == io.EOF {
				return tokens, i, nil
			}
			return tokens, i, err
		}
		tokens = append(tokens, t)
	}
	return
}

func mustTokenize(exp string) []token {
	tokens, _, err := tokenize(exp, 0)
	if err != nil {
		panic(err)
	}
	return tokens
}
