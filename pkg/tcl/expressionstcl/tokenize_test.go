// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//     https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package expressionstcl

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func slice(v ...interface{}) []interface{} {
	return v
}

func TestTokenizeSimple(t *testing.T) {
	operators := []string{"&&", "||", "!=", "<>", "==", "=", "+", "-", "*", ">", "<", "<=", ">=", "%", "**"}
	for _, op := range operators {
		assert.Equal(t, []token{tokenAccessor("a"), tokenMath(op), tokenAccessor("b")}, mustTokenize("a"+op+"b"))
	}
	assert.Equal(t, []token{tokenNot, tokenAccessor("abc")}, mustTokenize(`!abc`))
	assert.Equal(t, []token{tokenAccessor("a"), tokenTernary, tokenAccessor("b"), tokenTernarySeparator, tokenAccessor("c")}, mustTokenize(`a ? b : c`))
	assert.Equal(t, []token{tokenOpen, tokenAccessor("a"), tokenClose}, mustTokenize(`(a)`))
	assert.Equal(t, []token{tokenAccessor("a"), tokenOpen, tokenAccessor("b"), tokenComma, tokenJson(true), tokenClose}, mustTokenize(`a(b, true)`))
	assert.Equal(t, []token{tokenJson(noneValue)}, mustTokenize("null"))
	assert.Equal(t, []token{tokenJson(noneValue), tokenMath("+"), tokenJson(4.0)}, mustTokenize("null + 4"))
}

func TestTokenizeJson(t *testing.T) {
	assert.Equal(t, []token{tokenJson(1.0), tokenMath("+"), tokenJson(255.0)}, mustTokenize(`1 + 255`))
	assert.Equal(t, []token{tokenJson(1.6), tokenMath("+"), tokenJson(255.0)}, mustTokenize(`1.6 + 255`))
	assert.Equal(t, []token{tokenJson("abc"), tokenMath("+"), tokenJson("d")}, mustTokenize(`"abc" + "d"`))
	assert.Equal(t, []token{tokenJson(map[string]interface{}{"key1": "value1", "key2": "value2"})}, mustTokenize(`{"key1": "value1", "key2": "value2"}`))
	assert.Equal(t, []token{tokenJson(slice("a", "b"))}, mustTokenize(`["a", "b"]`))
	assert.Equal(t, []token{tokenJson(true)}, mustTokenize(`true`))
	assert.Equal(t, []token{tokenJson(false)}, mustTokenize(`false`))
}

func TestTokenizeComplex(t *testing.T) {
	want := []token{
		tokenAccessor("env.value"), tokenMath("&&"), tokenOpen, tokenAccessor("env.alternative"), tokenMath("+"), tokenJson("cd"), tokenClose, tokenMath("=="), tokenJson("abc"),
		tokenTernary, tokenJson(10.5),
		tokenTernarySeparator, tokenNot, tokenAccessor("ignored"), tokenTernary, tokenOpen, tokenJson(14.0), tokenMath("+"), tokenJson(3.1), tokenMath("*"), tokenJson(5.0), tokenClose,
		tokenTernarySeparator, tokenAccessor("transform"), tokenOpen, tokenJson("a"), tokenComma,
		tokenJson(map[string]interface{}{"x": "y"}), tokenComma, tokenJson(slice("z")), tokenClose,
	}
	assert.Equal(t, want, mustTokenize(`
		env.value && (env.alternative + "cd") == "abc"
			? 10.5
			: !ignored ? (14 + 3.1 * 5)
                       : transform("a",
                                   {"x": "y"}, ["z"])
	`))
}

func TestTokenizeInvalidAccessor(t *testing.T) {
	tokens, _, err := tokenize(`abc.`, 0)
	assert.Error(t, err)
	assert.Equal(t, []token{tokenAccessor("abc")}, tokens)
}

func TestTokenizeInvalidJson(t *testing.T) {
	tokens, _, err := tokenize(`{"abc": "d"`, 0)
	tokens2, _, err2 := tokenize(`{"abc": d}`, 0)
	assert.Error(t, err)
	assert.Equal(t, []token{}, tokens)
	assert.Error(t, err2)
	assert.Equal(t, []token{}, tokens2)
}
