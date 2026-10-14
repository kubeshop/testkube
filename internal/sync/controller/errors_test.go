package controller

import (
	"errors"
	"fmt"
	"testing"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	syncagent "github.com/kubeshop/testkube/internal/sync"
)

func TestTerminalOnRejection(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		wantTerminal bool
	}{
		{
			name:         "an ownership conflict stands until the owner changes",
			err:          fmt.Errorf("update TestWorkflow %q in store: %w", "smoke", syncagent.ErrOwnershipConflict),
			wantTerminal: true,
		},
		{
			name:         "an invalid resource stands until somebody changes it",
			err:          fmt.Errorf("update Webhook %q in store: %w", "hook", syncagent.ErrInvalidResource),
			wantTerminal: true,
		},
		{
			name: "any other failure is transient as far as the agent can tell",
			err:  errors.New("connection refused"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := terminalOnRejection(tt.err)

			if errors.Is(got, reconcile.TerminalError(nil)) != tt.wantTerminal {
				t.Errorf("terminal = %v, want %v, for %v", !tt.wantTerminal, tt.wantTerminal, got)
			}
			if !errors.Is(got, tt.err) {
				t.Errorf("expected the original error to stay in the chain, because the agent log shows only its message, got %v", got)
			}
		})
	}
}
