package expressions

import (
	"errors"
	"fmt"
	math2 "math"
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
			if priority >= 0 {
				return e, i, nil
			}
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
		case tokenTypePropertyAccessor:
			e = newPropertyAccessor(e, t[i].Value.(string))
			i += 1
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
		return NewValue(t[0].Value), 1, nil
	}

	// Negation - !expr
	if t[0].Type == tokenTypeNot {
		e, i, err = parseNextExpression(t[1:], math2.MaxInt)
		if err != nil {
			return nil, 0, err
		}
		return newNegative(e), i + 1, nil
	}

	// Negative numbers - -5
	if t[0].Type == tokenTypeMath && operator(t[0].Value.(string)) == operatorSubtract {
		e, i, err = parseNextExpression(t[1:], -1)
		if err != nil {
			return nil, 0, err
		}
		return newMath(operatorSubtract, NewValue(0), e), i + 1, nil
	}

	// Call - abc(a, b, c)
	if t[0].Type == tokenTypeAccessor && len(t) > 1 && t[1].Type == tokenTypeOpen {
		args := make([]CallArgument, 0)
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
			if len(t) > index && t[index].Type == tokenTypeSpread {
				args = append(args, CallArgument{Expression: next, Spread: true})
				index++
			} else {
				args = append(args, CallArgument{Expression: next})
			}
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
		return None, nil
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
	return v.Resolve()
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
			e = newMath(operatorAdd, e, NewStringValue(tpl[offset:offset+index]))
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
		v, err = v.Resolve()
		if err != nil {
			return nil, fmt.Errorf("expression error: %v", e)
		}
		e = newMath(operatorAdd, e, CastToString(v))
	}
	if offset < len(tpl) {
		e = newMath(operatorAdd, e, NewStringValue(tpl[offset:]))
	}
	if e == nil {
		return NewStringValue(""), nil
	}
	return e.Resolve()
}

func MustCompileTemplate(tpl string) Expression {
	v, err := CompileTemplate(tpl)
	if err != nil {
		panic(err)
	}
	return v
}

func CompileAndResolve(exp string, m ...Machine) (Expression, error) {
	e, err := Compile(exp)
	if err != nil {
		return e, err
	}
	return e.Resolve(m...)
}

func CompileAndResolveTemplate(tpl string, m ...Machine) (Expression, error) {
	e, err := CompileTemplate(tpl)
	if err != nil {
		return e, err
	}
	return e.Resolve(m...)
}

func IsTemplateStringWithoutExpressions(tpl string) bool {
	return !strings.Contains(tpl, "{{")
}

// ExtractPureTemplateExpression checks if a template string consists of only
// a single expression (i.e., "{{ expr }}") with no surrounding literal text.
// If so, it returns the inner expression string.
// This is used to detect cases where an expression may return a non-string
// value (like an array) that should be preserved rather than stringified.
func ExtractPureTemplateExpression(tpl string) (string, bool) {
	s := strings.TrimSpace(tpl)
	if !strings.HasPrefix(s, "{{") || !strings.HasSuffix(s, "}}") {
		return "", false
	}
	inner := s[2 : len(s)-2]
	if strings.Contains(inner, "{{") || strings.Contains(inner, "}}") {
		return "", false
	}
	inner = strings.TrimSpace(inner)
	if inner == "" {
		return "", false
	}
	return inner, true
}

// IsWildcardAccessorOnly checks whether an expression is purely a wildcard
// accessor chain (e.g., "services.slave.*.ip") with no additional operators,
// function calls, or array/object constructors.
//
// It also recognizes the compiled form of wildcard accessors — e.g.,
// _wc(services.slave,"_.value.ip") — which is the internal representation
// produced when the expression is compiled but cannot be fully resolved
// (because the accessor value is not yet known). This ensures that wildcard
// semantics (comma-join rather than array expansion) are preserved even after
// an expression round-trips through compile → serialize → re-compile.
//
// Such expressions resolve to arrays implicitly through the _wc() transform,
// but in template contexts they should be stringified (comma-joined) rather
// than expanded as separate slice elements.
//
// When a wildcard accessor is used inside an explicit array-producing construct
// (e.g., "list(services.slave.*.ip...)"), this function returns false so that
// expansion still works as expected.
func IsWildcardAccessorOnly(expr string) bool {
	tokens, _, _ := tokenize(expr, 0)
	if len(tokens) == 0 {
		return false
	}
	// The expression must consist of exactly one tokenTypeAccessor optionally
	// followed by one or more tokenTypePropertyAccessor tokens — nothing else.
	if tokens[0].Type == tokenTypeAccessor {
		allPropertyAccessors := true
		for _, tok := range tokens[1:] {
			if tok.Type != tokenTypePropertyAccessor {
				allPropertyAccessors = false
				break
			}
		}
		if allPropertyAccessors {
			// Now verify that the accessor chain actually contains a wildcard segment.
			for _, tok := range tokens {
				switch tok.Type {
				case tokenTypeAccessor:
					if name, ok := tok.Value.(string); ok {
						// Strip all whitespace so spaced accessors like
						// "services.slave . * . ip" are normalized to
						// "services.slave.*.ip" before the check.
						normalized := strings.Join(strings.Fields(name), "")
						if strings.Contains(normalized, ".*") {
							return true
						}
					}
				case tokenTypePropertyAccessor:
					if name, ok := tok.Value.(string); ok && strings.TrimSpace(name) == "*" {
						return true
					}
				}
			}
		}
	}

	// Check the compiled form: wildcard accessors are compiled into
	// _wc(<accessor>, "_.value[.<suffix>]") calls. Avoid the expensive
	// Compile/AST walk unless the raw expression starts with the internal
	// function name.
	if !couldBeCompiledWildcard(expr) {
		return false
	}
	return isCompiledWildcard(expr)
}

// couldBeCompiledWildcard performs a cheap pre-check: the expression must
// start with the internal wildcard function name followed by '('.
func couldBeCompiledWildcard(expr string) bool {
	trimmed := strings.TrimSpace(expr)
	if !strings.HasPrefix(trimmed, wildcardMapFn) {
		return false
	}
	rest := strings.TrimLeft(trimmed[len(wildcardMapFn):], " \t\r\n")
	return rest != "" && rest[0] == '('
}

// isCompiledWildcard returns true if expr is the compiled form of a
// wildcard accessor, i.e. _wc(<base>, "_.value[.<suffix>]") where <base>
// is either a plain accessor or a nested compiled wildcard.
func isCompiledWildcard(expr string) bool {
	compiled, err := Compile(expr)
	if err != nil {
		return false
	}
	return isWildcardMapExpr(compiled)
}

// isWildcardMapExpr checks if the given expression tree is a _wc() call
// produced by compiling a wildcard accessor.
func isWildcardMapExpr(expr Expression) bool {
	c, ok := expr.(*call)
	if !ok || c.name != wildcardMapFn || len(c.args) != 2 {
		return false
	}
	// Second argument must be a static string starting with "_.value"
	// (the compiler always produces "_.value" or "_.value.<suffix>").
	if c.args[1].Static() == nil {
		return false
	}
	s, err := c.args[1].Static().StringValue()
	if err != nil || (s != "_.value" && !strings.HasPrefix(s, "_.value.")) {
		return false
	}
	// First argument must be either a plain accessor or another _wc call
	switch c.args[0].Expression.(type) {
	case *accessor:
		return true
	case *call:
		return isWildcardMapExpr(c.args[0].Expression)
	}
	return false
}
