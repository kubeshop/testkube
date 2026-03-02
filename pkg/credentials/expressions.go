package credentials

import (
	"context"
	"fmt"

	"github.com/kubeshop/testkube/pkg/expressions"
)

const (
	SourceCredential = "credential"
	SourceVault      = "vault"
)

func NewCredentialMachine(repository CredentialRepository, observers ...func(name string, value string)) expressions.Machine {
	fetchCredentialWithSource := func(name string, computed bool, source string) (interface{}, error) {
		value, err := repository.GetWithSource(context.Background(), name, source)
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
			result, err := fetchCredentialWithSource(name, computed, SourceCredential)
			return result, true, err
		}).
		RegisterFunction("vault", func(values ...expressions.StaticValue) (interface{}, bool, error) {
			if len(values) != 1 {
				return nil, true, fmt.Errorf(`"vault" function expects 1 argument, %d provided`, len(values))
			}

			path, _ := values[0].StringValue()
			result, err := fetchCredentialWithSource(path, false, SourceVault)
			return result, true, err
		})
}
