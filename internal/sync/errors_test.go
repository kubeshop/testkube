package sync

import (
	"errors"
	"fmt"
	"testing"
)

func TestIsRejection(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "an ownership conflict never clears by retry", err: fmt.Errorf("sync: %w", ErrOwnershipConflict), want: true},
		{name: "an invalid resource never clears by retry", err: fmt.Errorf("sync: %w", ErrInvalidResource), want: true},
		{name: "any other failure is retried", err: errors.New("connection refused")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRejection(tt.err); got != tt.want {
				t.Errorf("IsRejection() = %v, want %v", got, tt.want)
			}
		})
	}
}
