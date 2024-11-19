package credentials

import (
	"context"
	"fmt"

	"github.com/kubeshop/testkube/pkg/expressions"
)

func NewCredentialMachine(repository CredentialRepository) expressions.Machine {
	return expressions.NewMachine().RegisterFunction("credential", func(values ...expressions.StaticValue) (interface{}, bool, error) {
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
		value, err := repository.Get(context.Background(), name)
		if err != nil {
			return nil, true, err
		}
		if computed {
			expr, err := expressions.CompileAndResolveTemplate(string(value))
			return expr, true, err
		}
		return string(value), true, nil
	})
}
