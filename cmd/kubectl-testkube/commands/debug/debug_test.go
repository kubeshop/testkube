package debug

import (
	"testing"
)

func TestBuildInfo(t *testing.T) {
	tests := []struct {
		name      string
		debugInfo DebugInfo
		want      string
		wantErr   bool
	}{
		{
			name:      "Empty DebugInfo",
			debugInfo: DebugInfo{},
			wantErr:   true,
		},
		{
			name: "Debug info populated",
			debugInfo: DebugInfo{
				ClientVersion:  "v0.test",
				ServerVersion:  "v1.test",
				Commit:         "commitID",
				BuildBy:        "test",
				BuildDate:      "now",
				ClusterVersion: "v2.test",
				APILogs:        []string{"api logline1", "api logline2"},
				OperatorLogs:   []string{"operator logline1", "operator logline2", "operator logline3"},
				LastExecutionLogs: map[string][]string{
					"execution1": {"execution logline1"},
					"execution2": {"execution logline1", "execution logline2"},
				},
			},
			want: `
|Property|Value|
|----|----|
|Client version|v0.test|
|Server version|v1.test|
|Commit|commitID|
|Build by|test|
|Build date|now|
|Cluster version|v2.test|

### API logs

api logline1
api logline2

### Operator logs

operator logline1
operator logline2
operator logline3

### Execution logs

Execution ID: execution1

execution logline1

Execution ID: execution2

execution logline1
execution logline2

`,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildInfo(tt.debugInfo)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("BuildInfo() = %v, want %v", got, tt.want)
			}
		})
	}
}
