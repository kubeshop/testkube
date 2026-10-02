package expressions

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccessor_Resolve_Errors(t *testing.T) {
	failing := NewMachine().RegisterAccessorExt(func(name string) (interface{}, bool, error) {
		return nil, true, errors.New("boom")
	})
	tests := []struct {
		name          string
		machine       Machine
		want          string
		wantUndefined bool
	}{
		{
			name:          "a variable that no machine defines",
			machine:       FinalizerFail,
			want:          "config.message is not defined",
			wantUndefined: true,
		},
		{
			name:    "a machine that fails keeps its error",
			machine: failing,
			want:    "error while accessing config.message: boom",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := Compile("config.message")
			require.NoError(t, err)
			_, err = expr.Resolve(tt.machine)
			assert.EqualError(t, err, tt.want)
			assert.Equal(t, tt.wantUndefined, errors.Is(err, ErrUnknownVariable))
		})
	}
}
