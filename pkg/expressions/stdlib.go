package expressions

import (
	"context"
	"encoding/json"
	"fmt"
	math2 "math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/itchyny/gojq"
	"github.com/kballard/go-shellquote"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

const (
	RFC3339Millis = "2006-01-02T15:04:05.000Z07:00"
)

type BasicFunctionHandler func(...StaticValue) (Expression, error)
type StdFunctionHandler func([]CallArgument) (Expression, bool, error)
type StdFunction struct {
	ReturnType Type
	Handler    StdFunctionHandler
}

type stdMachine struct{}

var StdLibMachine = &stdMachine{}

func ToStdFunctionHandler(fn BasicFunctionHandler) StdFunctionHandler {
	return func(args []CallArgument) (Expression, bool, error) {
		if !areArgsResolved(args) {
			return nil, false, nil
		}
		res, err := resolveArgs(args)
		if err != nil {
			return nil, true, err
		}
		expr, err := fn(res...)
		return expr, true, err
	}
}

var stdFunctions = map[string]StdFunction{
	"string": {
		ReturnType: TypeString,
		Handler: func(args []CallArgument) (Expression, bool, error) {
			if len(args) == 1 && !args[0].Spread && args[0].Type() == TypeString {
				return args[0].Expression, true, nil
			}
			if !areArgsResolved(args) {
				return nil, false, nil
			}
			value, err := resolveArgs(args)
			if err != nil {
				return nil, true, err
			}
			str := ""
			for i := range value {
				next, _ := value[i].StringValue()
				str += next
			}
			return NewValue(str), true, nil
		},
	},
	"list": {
		Handler: ToStdFunctionHandler(func(value ...StaticValue) (Expression, error) {
			v := make([]interface{}, len(value))
			for i := range value {
				v[i] = value[i].Value()
			}
			return NewValue(v), nil
		}),
	},
	"join": {
		ReturnType: TypeString,
		Handler: ToStdFunctionHandler(func(value ...StaticValue) (Expression, error) {
			if len(value) == 0 || len(value) > 2 {
				return nil, fmt.Errorf(`"join" function expects 1-2 arguments, %d provided`, len(value))
			}
			if value[0].IsNone() {
				return value[0], nil
			}
			if !value[0].IsSlice() {
				return nil, fmt.Errorf(`"join" function expects a slice as 1st argument: %v provided`, value[0].Value())
			}
			slice, err := value[0].SliceValue()
			if err != nil {
				return nil, fmt.Errorf(`"join" function error: reading slice: %s`, err.Error())
			}
			v := make([]string, len(slice))
			for i := range slice {
				v[i], _ = toString(slice[i])
			}
			separator := ","
			if len(value) == 2 {
				separator, _ = value[1].StringValue()
			}
			return NewValue(strings.Join(v, separator)), nil
		}),
	},
	"split": {
		Handler: ToStdFunctionHandler(func(value ...StaticValue) (Expression, error) {
			if len(value) == 0 || len(value) > 2 {
				return nil, fmt.Errorf(`"split" function expects 1-2 arguments, %d provided`, len(value))
			}
			str, _ := value[0].StringValue()
			separator := ","
			if len(value) == 2 {
				separator, _ = value[1].StringValue()
			}
			return NewValue(strings.Split(str, separator)), nil
		}),
	},
	"int": {
		ReturnType: TypeInt64,
		Handler: func(args []CallArgument) (Expression, bool, error) {
			if len(args) == 1 && !args[0].Spread && args[0].Type() == TypeInt64 {
				return args[0].Expression, true, nil
			}
			if !areArgsResolved(args) {
				return nil, false, nil
			}
			value, err := resolveArgs(args)
			if err != nil {
				return nil, true, err
			}
			if len(value) != 1 {
				return nil, true, fmt.Errorf(`"int" function expects 1 argument, %d provided`, len(value))
			}
			v, err := value[0].IntValue()
			if err != nil {
				return nil, true, err
			}
			return NewValue(v), true, nil
		},
	},
	"bool": {
		ReturnType: TypeBool,
		Handler: func(args []CallArgument) (Expression, bool, error) {
			if len(args) == 1 && !args[0].Spread && args[0].Type() == TypeBool {
				return args[0].Expression, true, nil
			}
			if !areArgsResolved(args) {
				return nil, false, nil
			}
			value, err := resolveArgs(args)
			if err != nil {
				return nil, true, err
			}
			if len(value) != 1 {
				return nil, true, fmt.Errorf(`"bool" function expects 1 argument, %d provided`, len(value))
			}
			v, err := value[0].BoolValue()
			if err != nil {
				return nil, true, err
			}
			return NewValue(v), true, nil
		},
	},
	"float": {
		ReturnType: TypeFloat64,
		Handler: func(args []CallArgument) (Expression, bool, error) {
			if len(args) == 1 && !args[0].Spread && args[0].Type() == TypeFloat64 {
				return args[0].Expression, true, nil
			}
			if !areArgsResolved(args) {
				return nil, false, nil
			}
			value, err := resolveArgs(args)
			if err != nil {
				return nil, true, err
			}
			if len(value) != 1 {
				return nil, true, fmt.Errorf(`"float" function expects 1 argument, %d provided`, len(value))
			}
			v, err := value[0].FloatValue()
			if err != nil {
				return nil, true, err
			}
			return NewValue(v), true, nil
		},
	},
	"tojson": {
		ReturnType: TypeString,
		Handler: ToStdFunctionHandler(func(value ...StaticValue) (Expression, error) {
			if len(value) != 1 {
				return nil, fmt.Errorf(`"tojson" function expects 1 argument, %d provided`, len(value))
			}
			b, err := json.Marshal(value[0].Value())
			if err != nil {
				return nil, fmt.Errorf(`"tojson" function had problem marshalling: %s`, err.Error())
			}
			return NewValue(string(b)), nil
		}),
	},
	"json": {
		Handler: ToStdFunctionHandler(func(value ...StaticValue) (Expression, error) {
			if len(value) != 1 {
				return nil, fmt.Errorf(`"json" function expects 1 argument, %d provided`, len(value))
			}
			if !value[0].IsString() {
				return nil, fmt.Errorf(`"json" function argument should be a string`)
			}
			var v interface{}
			err := json.Unmarshal([]byte(value[0].Value().(string)), &v)
			if err != nil {
				return nil, fmt.Errorf(`"json" function had problem unmarshalling: %s`, err.Error())
			}
			return NewValue(v), nil
		}),
	},
	"toyaml": {
		ReturnType: TypeString,
		Handler: ToStdFunctionHandler(func(value ...StaticValue) (Expression, error) {
			if len(value) != 1 {
				return nil, fmt.Errorf(`"toyaml" function expects 1 argument, %d provided`, len(value))
			}
			b, err := yaml.Marshal(value[0].Value())
			if err != nil {
				return nil, fmt.Errorf(`"toyaml" function had problem marshalling: %s`, err.Error())
			}
			return NewValue(string(b)), nil
		}),
	},
	"yaml": {
		Handler: ToStdFunctionHandler(func(value ...StaticValue) (Expression, error) {
			if len(value) != 1 {
				return nil, fmt.Errorf(`"yaml" function expects 1 argument, %d provided`, len(value))
			}
			if !value[0].IsString() {
				return nil, fmt.Errorf(`"yaml" function argument should be a string`)
			}
			var v interface{}
			err := yaml.Unmarshal([]byte(value[0].Value().(string)), &v)
			if err != nil {
				return nil, fmt.Errorf(`"yaml" function had problem unmarshalling: %s`, err.Error())
			}
			return NewValue(v), nil
		}),
	},
	"shellquote": {
		ReturnType: TypeString,
		Handler: ToStdFunctionHandler(func(value ...StaticValue) (Expression, error) {
			args := make([]string, len(value))
			for i := range value {
				args[i], _ = value[i].StringValue()
			}
			return NewValue(shellquote.Join(args...)), nil
		}),
	},
	"shellparse": {
		Handler: ToStdFunctionHandler(func(value ...StaticValue) (Expression, error) {
			if len(value) != 1 {
				return nil, fmt.Errorf(`"shellparse" function expects 1 arguments, %d provided`, len(value))
			}
			v, _ := value[0].StringValue()
			words, err := shellquote.Split(v)
			return NewValue(words), err
		}),
	},
	"trim": {
		ReturnType: TypeString,
		Handler: ToStdFunctionHandler(func(value ...StaticValue) (Expression, error) {
			if len(value) != 1 {
				return nil, fmt.Errorf(`"trim" function expects 1 argument, %d provided`, len(value))
			}
			if !value[0].IsString() {
				return nil, fmt.Errorf(`"trim" function argument should be a string`)
			}
			str, _ := value[0].StringValue()
			return NewValue(strings.TrimSpace(str)), nil
		}),
	},
	"len": {
		ReturnType: TypeInt64,
		Handler: ToStdFunctionHandler(func(value ...StaticValue) (Expression, error) {
			if len(value) != 1 {
				return nil, fmt.Errorf(`"len" function expects 1 argument, %d provided`, len(value))
			}
			if value[0].IsSlice() {
				v, err := value[0].SliceValue()
				return NewValue(int64(len(v))), err
			}
			if value[0].IsString() {
				v, err := value[0].StringValue()
				return NewValue(int64(len(v))), err
			}
			if value[0].IsMap() {
				v, err := value[0].MapValue()
				return NewValue(int64(len(v))), err
			}
			return nil, fmt.Errorf(`"len" function expects string, slice or map, %v provided`, value[0])
		}),
	},
	"floor": {
		ReturnType: TypeInt64,
		Handler: ToStdFunctionHandler(func(value ...StaticValue) (Expression, error) {
			if len(value) != 1 {
				return nil, fmt.Errorf(`"floor" function expects 1 argument, %d provided`, len(value))
			}
			f, err := value[0].FloatValue()
			if err != nil {
				return nil, fmt.Errorf(`"floor" function expects a number, %s provided: %v`, value[0], err)
			}
			return NewValue(int64(math2.Floor(f))), nil
		}),
	},
	"ceil": {
		ReturnType: TypeInt64,
		Handler: ToStdFunctionHandler(func(value ...StaticValue) (Expression, error) {
			if len(value) != 1 {
				return nil, fmt.Errorf(`"ceil" function expects 1 argument, %d provided`, len(value))
			}
			f, err := value[0].FloatValue()
			if err != nil {
				return nil, fmt.Errorf(`"ceil" function expects a number, %s provided: %v`, value[0], err)
			}
			return NewValue(int64(math2.Ceil(f))), nil
		}),
	},
	"round": {
		ReturnType: TypeInt64,
		Handler: ToStdFunctionHandler(func(value ...StaticValue) (Expression, error) {
			if len(value) != 1 {
				return nil, fmt.Errorf(`"round" function expects 1 argument, %d provided`, len(value))
			}
			f, err := value[0].FloatValue()
			if err != nil {
				return nil, fmt.Errorf(`"round" function expects a number, %s provided: %v`, value[0], err)
			}
			return NewValue(int64(math2.Round(f))), nil
		}),
	},
	"chunk": {
		Handler: ToStdFunctionHandler(func(value ...StaticValue) (Expression, error) {
			if len(value) != 2 {
				return nil, fmt.Errorf(`"chunk" function expects 2 arguments, %d provided`, len(value))
			}
			list, err := value[0].SliceValue()
			if err != nil {
				return nil, fmt.Errorf(`"chunk" function expects 1st argument to be a list, %s provided: %v`, value[0], err)
			}
			size, err := value[1].IntValue()
			if err != nil {
				return nil, fmt.Errorf(`"chunk" function expects 2nd argument to be integer, %s provided: %v`, value[1], err)
			}
			if size <= 0 {
				return nil, fmt.Errorf(`"chunk" function expects 2nd argument to be >= 1, %s provided: %v`, value[1], err)
			}
			chunks := make([][]interface{}, 0)
			l := int64(len(list))
			for i := int64(0); i < l; i += size {
				end := i + size
				if end > l {
					end = l
				}
				chunks = append(chunks, list[i:end])
			}
			return NewValue(chunks), nil
		}),
	},
	"at": {
		Handler: ToStdFunctionHandler(func(value ...StaticValue) (Expression, error) {
			if len(value) != 2 {
				return nil, fmt.Errorf(`"at" function expects 2 arguments, %d provided`, len(value))
			}
			if value[0].IsSlice() {
				v, _ := value[0].SliceValue()
				k, err := value[1].IntValue()
				if err != nil {
					return nil, fmt.Errorf(`"at" function expects 2nd argument to be number for list, %s provided`, value[1])
				}
				if k >= 0 && k < int64(len(v)) {
					return NewValue(v[int(k)]), nil
				}
				return nil, fmt.Errorf(`"at" function: error: out of bounds (length=%d, index=%d)`, len(v), k)
			}
			if value[0].IsMap() {
				v, _ := value[0].MapValue()
				k, _ := value[1].StringValue()
				item, ok := v[k]
				if ok {
					return NewValue(item), nil
				}
				return None, nil
			}
			if value[0].IsString() {
				v, _ := value[0].StringValue()
				k, err := value[1].IntValue()
				if err != nil {
					return nil, fmt.Errorf(`"at" function expects 2nd argument to be number for string, %s provided`, value[1])
				}
				if k >= 0 && k < int64(len(v)) {
					return NewValue(v[int(k)]), nil
				}
				return nil, fmt.Errorf(`"at" function: error: out of bounds (length=%d, index=%d)`, len(v), k)
			}
			return nil, fmt.Errorf(`"at" function can be performed only on lists, maps and strings: %s provided`, value[0])
		}),
	},
	"map": {
		Handler: ToStdFunctionHandler(func(value ...StaticValue) (Expression, error) {
			if len(value) != 2 {
				return nil, fmt.Errorf(`"map" function expects 2 arguments, %d provided`, len(value))
			}
			list, err := value[0].SliceValue()
			if err != nil {
				return nil, fmt.Errorf(`"map" function expects 1st argument to be a list, %s provided: %v`, value[0], err)
			}
			exprStr, _ := value[1].StringValue()
			expr, err := Compile(exprStr)
			if err != nil {
				return nil, fmt.Errorf(`"map" function expects 2nd argument to be valid expression, '%s' provided: %v`, value[1], err)
			}
			result := make([]string, len(list))
			for i := 0; i < len(list); i++ {
				ex, _ := Compile(expr.String())
				v, err := ex.Resolve(NewMachine().Register("_.value", list[i]).Register("_.index", i).Register("_.key", i))
				if err != nil {
					return nil, fmt.Errorf(`"map" function: error while mapping %d index (%v): %v`, i, list[i], err)
				}
				result[i] = v.String()
			}
			return Compile(fmt.Sprintf("list(%s)", strings.Join(result, ",")))
		}),
	},
	"entries": {
		Handler: ToStdFunctionHandler(func(value ...StaticValue) (Expression, error) {
			if len(value) != 1 {
				return nil, fmt.Errorf(`"entries" function expects 1 argument, %d provided`, len(value))
			}
			dict, err := value[0].MapValue()
			if err != nil {
				return nil, fmt.Errorf(`"entries" function expects 1st argument to be a map, %s provided: %v`, value[0], err)
			}
			list := make([]MapEntry, 0, len(dict))
			for k, v := range dict {
				list = append(list, MapEntry{Key: k, Value: v})
			}
			return NewValue(list), nil
		}),
	},
	"filter": {
		Handler: ToStdFunctionHandler(func(value ...StaticValue) (Expression, error) {
			if len(value) != 2 {
				return nil, fmt.Errorf(`"filter" function expects 2 arguments, %d provided`, len(value))
			}
			list, err := value[0].SliceValue()
			if err != nil {
				return nil, fmt.Errorf(`"filter" function expects 1st argument to be a list, %s provided: %v`, value[0], err)
			}
			exprStr, _ := value[1].StringValue()
			expr, err := Compile(exprStr)
			if err != nil {
				return nil, fmt.Errorf(`"filter" function expects 2nd argument to be valid expression, '%s' provided: %v`, value[1], err)
			}
			result := make([]interface{}, 0)
			for i := 0; i < len(list); i++ {
				ex, _ := Compile(expr.String())
				v, err := ex.Resolve(NewMachine().Register("_.value", list[i]).Register("_.index", i).Register("_.key", i))
				if err != nil {
					return nil, fmt.Errorf(`"filter" function: error while filtering %d index (%v): %v`, i, list[i], err)
				}
				if v.Static() == nil {
					// TODO: It shouldn't fail then
					return nil, fmt.Errorf(`"filter" function: could not resolve filter for %d index (%v): %s`, i, list[i], v)
				}
				b, err := v.Static().BoolValue()
				if err != nil {
					return nil, fmt.Errorf(`"filter" function: could not resolve filter for %d index (%v) as boolean: %s`, i, list[i], err)
				}
				if b {
					result = append(result, list[i])
				}
			}
			return NewValue(result), nil
		}),
	},
	"eval": {
		Handler: ToStdFunctionHandler(func(value ...StaticValue) (Expression, error) {
			if len(value) != 1 {
				return nil, fmt.Errorf(`"eval" function expects 1 argument, %d provided`, len(value))
			}
			exprStr, _ := value[0].StringValue()
			expr, err := Compile(exprStr)
			if err != nil {
				return nil, fmt.Errorf(`"eval" function: %s: error: %v`, value[0], err)
			}
			return expr, nil
		}),
	},
	"jq": {
		Handler: ToStdFunctionHandler(func(value ...StaticValue) (Expression, error) {
			if len(value) != 2 {
				return nil, fmt.Errorf(`"jq" function expects 2 arguments, %d provided`, len(value))
			}
			queryStr, _ := value[1].StringValue()
			query, err := gojq.Parse(queryStr)
			if err != nil {
				return nil, fmt.Errorf(`"jq" error: could not parse the query: %s: %v`, queryStr, err)
			}

			// Marshal data to basic types
			bytes, err := json.Marshal(value[0].Value())
			if err != nil {
				return nil, fmt.Errorf(`"jq" error: could not marshal the value: %v: %v`, value[0].Value(), err)
			}
			var v interface{}
			_ = json.Unmarshal(bytes, &v)

			// Run query against the value
			ctx, ctxCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer ctxCancel()
			iter := query.RunWithContext(ctx, v)
			result := make([]interface{}, 0)
			for {
				v, ok := iter.Next()
				if !ok {
					break
				}
				if err, ok := v.(error); ok {
					return nil, errors.Wrap(err, `"jq" error: executing: %v`)
				}
				result = append(result, v)
			}
			return NewValue(result), nil
		}),
	},
	"relpath": {
		ReturnType: TypeString,
		Handler: ToStdFunctionHandler(func(value ...StaticValue) (Expression, error) {
			if len(value) != 1 && len(value) != 2 {
				return nil, fmt.Errorf(`"relpath" function expects 1-2 arguments, %d provided`, len(value))
			}
			destinationPath, _ := value[0].StringValue()
			sourcePath := "/"
			if len(value) == 2 {
				sourcePath, _ = value[1].StringValue()
			} else {
				cwd, err := os.Getwd()
				if err == nil {
					sourcePath = cwd
				}
			}
			destinationPath, err := filepath.Abs(destinationPath)
			if err != nil {
				return nil, err
			}
			sourcePath, err = filepath.Abs(sourcePath)
			if err != nil {
				return nil, err
			}
			v, err := filepath.Rel(sourcePath, destinationPath)
			return NewValue(v), err
		}),
	},
	"abspath": {
		ReturnType: TypeString,
		Handler: ToStdFunctionHandler(func(value ...StaticValue) (Expression, error) {
			if len(value) != 1 && len(value) != 2 {
				return nil, fmt.Errorf(`"abspath" function expects 1-2 arguments, %d provided`, len(value))
			}
			destinationPath, _ := value[0].StringValue()
			if filepath.IsAbs(destinationPath) {
				return NewValue(filepath.Clean(destinationPath)), nil
			}

			sourcePath := "/"
			if len(value) == 2 {
				sourcePath, _ = value[1].StringValue()
			} else {
				cwd, err := os.Getwd()
				if err == nil {
					sourcePath = cwd
				}
			}
			sourcePath, err := filepath.Abs(sourcePath)
			if err != nil {
				return nil, err
			}
			return NewValue(filepath.Join(sourcePath, destinationPath)), err
		}),
	},
	"range": {
		Handler: ToStdFunctionHandler(func(value ...StaticValue) (Expression, error) {
			if len(value) != 1 && len(value) != 2 {
				return nil, fmt.Errorf(`"range" function expects 1-2 arguments, %d provided`, len(value))
			}

			// Compute start value
			start, err := value[0].IntValue()
			if err != nil {
				return nil, fmt.Errorf(`"range" function expects integer arguments, %s provided`, value[0].String())
			}

			// Compute end value
			var end int64
			if len(value) == 1 {
				end = start
				start = 0
			} else {
				end, err = value[1].IntValue()
				if err != nil {
					return nil, fmt.Errorf(`"range" function expects integer arguments, %s provided`, value[1].String())
				}
			}

			// Build a range (inclusive start, exclusive end)
			items := int64(math2.Max(0, float64(end-start)))
			result := make([]int64, items)
			for i := int64(0); i < items; i++ {
				result[i] = start + i
			}
			return NewValue(result), nil
		}),
	},
	"date": {
		ReturnType: TypeString,
		Handler: ToStdFunctionHandler(func(value ...StaticValue) (Expression, error) {
			if len(value) == 0 {
				return NewValue(time.Now().UTC().Format(RFC3339Millis)), nil
			} else if len(value) == 1 {
				format, _ := value[0].StringValue()
				return NewValue(time.Now().UTC().Format(format)), nil
			}
			return nil, fmt.Errorf(`"date" function expects 0-1 arguments, %d provided`, len(value))
		}),
	},
}

const (
	stringCastStdFn = "string"
	boolCastStdFn   = "bool"
	intCastStdFn    = "int"
	floatCastStdFn  = "float"
)

type MapEntry struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

func CastToString(v Expression) Expression {
	if v.Static() != nil {
		return NewStringValue(v.Static().Value())
	} else if v.Type() == TypeString {
		return v
	}
	return newCall(stringCastStdFn, []CallArgument{{Expression: v}})
}

func CastToBool(v Expression) Expression {
	if v.Type() == TypeBool {
		return v
	}
	return newCall(boolCastStdFn, []CallArgument{{Expression: v}})
}

func CastToInt(v Expression) Expression {
	if v.Type() == TypeInt64 {
		return v
	}
	return newCall(intCastStdFn, []CallArgument{{Expression: v}})
}

func CastToFloat(v Expression) Expression {
	if v.Type() == TypeFloat64 {
		return v
	}
	return newCall(intCastStdFn, []CallArgument{{Expression: v}})
}

func IsStdFunction(name string) bool {
	_, ok := stdFunctions[name]
	return ok
}

func GetStdFunctionReturnType(name string) Type {
	return stdFunctions[name].ReturnType
}

func CallStdFunction(name string, value ...interface{}) (Expression, error) {
	fn, ok := stdFunctions[name]
	if !ok {
		return nil, fmt.Errorf("function '%s' doesn't exists in standard library", name)
	}
	r := make([]CallArgument, 0, len(value))
	for i := 0; i < len(value); i++ {
		if v, ok := value[i].(StaticValue); ok {
			r = append(r, CallArgument{Expression: v})
		} else if v, ok := value[i].(Expression); ok {
			return nil, fmt.Errorf("expression functions can be called only with static values: %s provided", v)
		} else {
			r = append(r, CallArgument{Expression: NewValue(value[i])})
		}
	}
	v, _, err := fn.Handler(r)
	return v, err
}

func (*stdMachine) Get(name string) (Expression, bool, error) {
	return nil, false, nil
}

func (*stdMachine) Call(name string, args []CallArgument) (Expression, bool, error) {
	fn, ok := stdFunctions[name]
	if !ok {
		return nil, false, nil
	}
	return fn.Handler(args)
}
