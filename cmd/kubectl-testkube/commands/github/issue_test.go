package github

import (
	"testing"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func TestBuildTicket(t *testing.T) {
	tests := []struct {
		name      string
		debugInfo testkube.DebugInfo
		wantTitle string
		wantBody  string
		wantErr   bool
	}{
		{
			name:      "Empty DebugInfo",
			debugInfo: testkube.DebugInfo{},
			wantErr:   true,
		},
		{
			name: "Debug info populated",
			debugInfo: testkube.DebugInfo{
				ClientVersion:  "v0.test",
				ServerVersion:  "v1.test",
				ClusterVersion: "v2.test",
				ApiLogs:        []string{"api logline1", "api logline2"},
				OperatorLogs:   []string{"operator logline1", "operator logline2", "operator logline3"},
				ExecutionLogs: map[string][]string{
					"execution1": {"execution logline1"},
					"execution2": {"execution logline1", "execution logline2"},
				},
			},
			wantTitle: "New bug report",
			wantBody: `
**Describe the bug**
A clear and concise description of what the bug is.

**To Reproduce**
Steps to reproduce the behavior:
1. Run '...'
2. Specify '...'
3. See error

**Expected behavior**
A clear and concise description of what you expected to happen.

**Version / Cluster**
- Testkube CLI version: v0.test
- Testkube API server version: v1.test
- Kubernetes cluster version: v2.test

**Screenshots**
If applicable, add CLI commands/output to help explain your problem.

**Additional context**
Add any other context about the problem here.

Attach the output of the **testkube debug info** command to provide more details.
`,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTitle, gotBody, err := buildTicket(tt.debugInfo)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildTicket() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotTitle != tt.wantTitle {
				t.Errorf("BuildTicket() title = %v, want %v", gotTitle, tt.wantTitle)
			}
			if gotBody != tt.wantBody {
				t.Errorf("BuildTicket() body = %v, want %v", gotBody, tt.wantBody)
			}
		})
	}
}
