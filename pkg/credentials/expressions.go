package credentials

import (
	"context"
	"fmt"

	"github.com/kubeshop/testkube/pkg/expressions"
)

func NewCredentialMachine(repository CredentialRepository, observers ...func(name string, value string)) expressions.Machine {
	// Helper function to handle credential fetching logic
	fetchCredential := func(name string, computed bool) (interface{}, error) {
		value, err := repository.Get(context.Background(), name)
		if err != nil {
			return nil, err
		}
		if computed {
			expr, err := expressions.CompileAndResolveTemplate(string(value))
			// TODO: consider obfuscating each static part, if it's not finalized yet
			if expr.Static() != nil {
				strValue, _ := expr.Static().StringValue()
				for i := range observers {
					observers[i](name, strValue)
				}
			}
			return expr, err
		}
		valueStr := string(value)
		for i := range observers {
			observers[i](name, valueStr)
		}
		return valueStr, nil
	}

	return expressions.NewMachine().
		RegisterFunction("credential", func(values ...expressions.StaticValue) (interface{}, bool, error) {
			computed := false
			if len(values) == 2 {
				if values[1].IsBool() {
					computed, _ = values[1].BoolValue()
				} else {
					return nil, true, fmt.Errorf(`"credential" function expects 2nd argument to be boolean, %s provided`, values[1].String())
				}
			} else if len(values) != 1 {
				return nil, true, fmt.Errorf(`"credential" function expects 1-2 arguments, %d provided`, len(values))
			}

			name, _ := values[0].StringValue()
			result, err := fetchCredential(name, computed)
			return result, true, err
		}).
		RegisterFunction("encrypted", func(values ...expressions.StaticValue) (interface{}, bool, error) {
			computed := false
			if len(values) == 2 {
				if values[1].IsBool() {
					computed, _ = values[1].BoolValue()
				} else {
					return nil, true, fmt.Errorf(`"encrypted" function expects 2nd argument to be boolean, %s provided`, values[1].String())
				}
			} else if len(values) != 1 {
				return nil, true, fmt.Errorf(`"encrypted" function expects 1-2 arguments, %d provided`, len(values))
			}

			name, _ := values[0].StringValue()
			result, err := fetchCredential(name, computed)
			return result, true, err
		}).
		RegisterFunction("variable", func(values ...expressions.StaticValue) (interface{}, bool, error) {
			computed := false
			if len(values) == 2 {
				if values[1].IsBool() {
					computed, _ = values[1].BoolValue()
				} else {
					return nil, true, fmt.Errorf(`"variable" function expects 2nd argument to be boolean, %s provided`, values[1].String())
				}
			} else if len(values) != 1 {
				return nil, true, fmt.Errorf(`"variable" function expects 1-2 arguments, %d provided`, len(values))
			}

			name, _ := values[0].StringValue()
			result, err := fetchCredential(name, computed)
			return result, true, err
		}).
		RegisterFunction("vault", func(values ...expressions.StaticValue) (interface{}, bool, error) {
			computed := false
			if len(values) == 2 {
				if values[1].IsBool() {
					computed, _ = values[1].BoolValue()
				} else {
					return nil, true, fmt.Errorf(`"vault" function expects 2nd argument to be boolean, %s provided`, values[1].String())
				}
			} else if len(values) != 1 {
				return nil, true, fmt.Errorf(`"vault" function expects 1-2 arguments, %d provided`, len(values))
			}

			name, _ := values[0].StringValue()
			result, err := fetchCredential(name, computed)
			return result, true, err
		})
}
