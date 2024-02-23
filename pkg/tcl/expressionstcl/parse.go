// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//     https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package expressionstcl

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

func parseNextExpression(t []token, priority int) (e Expression, i int, err error) {
	e, i, err = getNextSegment(t)
	if err != nil {
		return
	}

	for {
		// End of the expression
		if len(t) == i {
			return e, i, nil
		}

		switch t[i].Type {
		case tokenTypeTernary:
			i += 1
			te, ti, terr := parseNextExpression(t[i:], 0)
			i += ti
			if terr != nil {
				return nil, i, terr
			}
			if len(t) == i {
				return nil, i, fmt.Errorf("premature end of expression: expected ternary separator")
			}
			if t[i].Type != tokenTypeTernarySeparator {
				return nil, i, fmt.Errorf("expression syntax error: expected ternary separator: found %v", t[i])
			}
			i++
			fe, fi, ferr := parseNextExpression(t[i:], 0)
			i += fi
			if ferr != nil {
				return nil, i, ferr
			}
			e = newConditional(e, te, fe)
		case tokenTypeMath:
			op := operator(t[i].Value.(string))
			nextPriority := getOperatorPriority(op)
			if priority >= nextPriority {
				return e, i, nil
			}
			i += 1
			ne, ni, nerr := parseNextExpression(t[i:], nextPriority)
			i += ni
			if nerr != nil {
				return nil, i, nerr
			}
			e = newMath(op, e, ne)
		default:
			return e, i, err
		}
	}
}

func getNextSegment(t []token) (e Expression, i int, err error) {
	if len(t) == 0 {
		return nil, 0, errors.New("premature end of expression")
	}

	// Parentheses - (a(b) + c)
	if t[0].Type == tokenTypeOpen {
		e, i, err = parseNextExpression(t[1:], -1)
		i++
		if err != nil {
			return nil, i, err
		}
		if len(t) <= i || t[i].Type != tokenTypeClose {
			return nil, i, fmt.Errorf("syntax error: expected parentheses close")
		}
		return e, i + 1, err
	}

	// Static value - "abc", 444, {"a": 10}, true, [45, 3]
	if t[0].Type == tokenTypeJson {
		return newStatic(t[0].Value), 1, nil
	}

	// Negation - !expr
	if t[0].Type == tokenTypeNot {
		e, i, err = parseNextExpression(t[1:], -1)
		if err != nil {
			return nil, 0, err
		}
		return newNegative(e), i + 1, nil
	}

	// Call - abc(a, b, c)
	if t[0].Type == tokenTypeAccessor && len(t) > 1 && t[1].Type == tokenTypeOpen {
		args := make([]Expression, 0)
		index := 2
		for {
			// Ensure there is another token (for call close or next argument)
			if len(t) <= index {
				return nil, 2, errors.New("premature end of expression: missing call close")
			}

			// Close the call
			if t[index].Type == tokenTypeClose {
				break
			}

			// Ensure comma between arguments
			if len(args) != 0 {
				if t[index].Type != tokenTypeComma {
					return nil, 2, errors.New("expression syntax error: expected comma or call close")
				}
				index++
			}
			next, l, err := parseNextExpression(t[index:], -1)
			index += l
			if err != nil {
				return nil, index, err
			}
			args = append(args, next)
		}
		return newCall(t[0].Value.(string), args), index + 1, nil
	}

	// Accessor - abc
	if t[0].Type == tokenTypeAccessor {
		return newAccessor(t[0].Value.(string)), 1, nil
	}

	return nil, 0, fmt.Errorf("unexpected token in expression: %v", t)
}

func parse(t []token) (e Expression, err error) {
	if len(t) == 0 {
		return newStatic(noneValue), nil
	}
	e, l, err := parseNextExpression(t, -1)
	if err != nil {
		return nil, err
	}
	if l < len(t) {
		return nil, fmt.Errorf("unexpected token after end of expression: %v", t[l])
	}
	return e, nil
}

func Compile(exp string) (Expression, error) {
	t, _, e := tokenize(exp, 0)
	if e != nil {
		return nil, fmt.Errorf("tokenizer error: %v", e)
	}
	v, e := parse(t)
	if e != nil {
		return nil, fmt.Errorf("parser error: %v", e)
	}
	return v.Simplify(nil)
}

func MustCompile(exp string) Expression {
	v, err := Compile(exp)
	if err != nil {
		panic(err)
	}
	return v
}

var endExprRe = regexp.MustCompile(`^\s*}}`)

func CompileTemplate(tpl string) (Expression, error) {
	var e Expression

	offset := 0
	for index := strings.Index(tpl[offset:], "{{"); index != -1; index = strings.Index(tpl[offset:], "{{") {
		if index != 0 {
			e = newMath(operatorAdd, e, newStaticString(tpl[offset:offset+index]))
		}
		offset += index + 2
		tokens, i, err := tokenize(tpl, offset)
		offset = i
		if err == nil {
			return nil, errors.New("template error: expression not closed")
		}
		if !endExprRe.MatchString(tpl[offset:]) || !strings.Contains(err.Error(), "unknown character") {
			return nil, fmt.Errorf("tokenizer error: %v", err)
		}
		offset += len(endExprRe.FindString(tpl[offset:]))
		if len(tokens) == 0 {
			continue
		}
		v, err := parse(tokens)
		if err != nil {
			return nil, fmt.Errorf("parser error: %v", e)
		}
		v, err = v.Simplify(nil)
		if err != nil {
			return nil, fmt.Errorf("expression error: %v", e)
		}
		if vv, ok := v.(StringAwareExpression); ok && vv.WillBeString() {
			e = newMath(operatorAdd, e, v)
		} else if v.Static() != nil {
			e = newMath(operatorAdd, e, newStaticString(v.Static().Value()))
		} else {
			e = newMath(operatorAdd, e, newCall("string", []Expression{v}))
		}
	}
	if offset < len(tpl) {
		e = newMath(operatorAdd, e, newStaticString(tpl[offset:]))
	}
	return e.Simplify(nil)
}

func MustCompileTemplate(tpl string) Expression {
	v, err := CompileTemplate(tpl)
	if err != nil {
		panic(err)
	}
	return v
}
