package telemetry

import (
	"runtime"
	"time"

	"github.com/kubeshop/testkube/pkg/utils/text"
)

// SendMCPToolEvent sends telemetry for MCP tool execution
func SendMCPToolEvent(toolName string, duration time.Duration, hasError bool, version string) (string, error) {
	payload := NewMCPToolPayload(toolName, "testkube_mcp_tool_execution", duration, version, hasError)
	return sendData(senders, payload)
}

// NewMCPToolPayload creates a payload for MCP tool telemetry events
func NewMCPToolPayload(toolName, eventName string, duration time.Duration, version string, hasError bool) Payload {
	machineID := GetMachineID()
	return Payload{
		ClientID: machineID,
		UserID:   machineID,
		Events: []Event{{
			Name: text.GAEventName(eventName),
			Params: Params{
				EventCount:      1,
				EventCategory:   "testkube_mcp_tool",
				AppVersion:      version,
				AppName:         "testkube-mcp-server",
				MachineID:       machineID,
				OperatingSystem: runtime.GOOS,
				Architecture:    runtime.GOARCH,
				Context:         getCurrentContext(),
				ClusterType:     GetClusterType(),
				Status: func() string {
					if hasError {
						return "error"
					} else {
						return "success"
					}
				}(),
				ToolName:   toolName,
				DurationMs: int32(duration.Milliseconds()),
				CliContext: GetCliRunContext(),
			},
		}},
	}
}
