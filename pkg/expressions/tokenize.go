package expressions

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
var accessorRe = regexp.MustCompile(`^[a-zA-Z\d_]+(?:\s*\.\s*([a-zA-Z\d_]+|\*))*`)
var propertyAccessorRe = regexp.MustCompile(`^\.\s*([a-zA-Z\d_]+|\*)`)
var spreadRe = regexp.MustCompile(`^\.\.\.`)
var spaceRe = regexp.MustCompile(`^\s+`)
var singleQuoteStringRe = regexp.MustCompile(`^'(\\'|[^'])*'`)

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
		case spreadRe.MatchString(exp[i:]):
			return tokenSpread, i + 3, nil
		case singleQuoteStringRe.MatchString(exp[i:]):
			// TODO: Optimize, and allow deeper in the tree (i.e. as part of array or object)
			str := singleQuoteStringRe.Find([]byte(exp[i:]))
			originalLen := len(str)
			for index := 1; index < len(str)-1; index++ {
				switch str[index] {
				case '\\':
					if len(str) > index+2 && str[index+1] == '\'' {
						str = append(str[0:index], str[index+1:]...)
					} else {
						index++
					}
				case '"':
					str = append(str[0:index], append([]byte{'\\', '"'}, str[index+1:]...)...)
					index++
				case '\n':
					str = append(str[0:index], append([]byte{'\\', 'n'}, str[index+1:]...)...)
					index++
				case '\t':
					str = append(str[0:index], append([]byte{'\\', 't'}, str[index+1:]...)...)
					index++
				}
			}
			str[0], str[len(str)-1] = '"', '"'
			decoder := json.NewDecoder(bytes.NewBuffer(str))
			var val interface{}
			err := decoder.Decode(&val)
			if err != nil {
				return token{}, i, fmt.Errorf("error while decoding string from index %d in expression: %s: %s", i, exp, err.Error())
			}
			return tokenJson(val), i + originalLen, nil
		case jsonValueRe.MatchString(exp[i:]):
			// Allow multi-line string with literal \n
			// TODO: Optimize, and allow deeper in the tree (i.e. as part of array or object)
			appended := 0
			if exp[i] == '"' {
				inside := true
				for index := i + 1; inside && index < len(exp); index++ {
					switch exp[index] {
					case '\\':
						index++
					case '"':
						inside = false
					case '\n':
						exp = exp[0:index] + "\\n" + exp[index+1:]
						appended++
					case '\t':
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
		case propertyAccessorRe.MatchString(exp[i:]):
			acc := propertyAccessorRe.FindString(exp[i:])
			return tokenPropertyAccessor(acc[1:]), i + len(acc), nil
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
