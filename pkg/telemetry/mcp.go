package telemetry

import (
	"runtime"
	"time"

	"github.com/kubeshop/testkube/pkg/utils/text"
)

// SendMCPToolEvent sends telemetry for MCP tool execution
func SendMCPToolEvent(toolName string, duration time.Duration, hasError bool, version string) (string, error) {
	eventName := "mcp_tool_execution"
	if hasError {
		eventName = "mcp_tool_error"
	}

	payload := NewMCPToolPayload(toolName, eventName, duration, version)
	return sendData(senders, payload)
}

// NewMCPToolPayload creates a payload for MCP tool telemetry events
func NewMCPToolPayload(toolName, eventName string, duration time.Duration, version string) Payload {
	machineID := GetMachineID()
	return Payload{
		ClientID: machineID,
		UserID:   machineID,
		Events: []Event{{
			Name: text.GAEventName(eventName),
			Params: Params{
				EventCount:      1,
				EventCategory:   "mcp_tool",
				AppVersion:      version,
				AppName:         "testkube-mcp",
				MachineID:       machineID,
				OperatingSystem: runtime.GOOS,
				Architecture:    runtime.GOARCH,
				Context:         getCurrentContext(),
				ClusterType:     GetClusterType(),
				ToolName:        toolName,
				DurationMs:      int32(duration.Milliseconds()),
				Status:          getStatusFromDuration(duration),
				CliContext:      GetCliRunContext(),
			},
		}},
	}
}

// getStatusFromDuration determines status based on execution duration
func getStatusFromDuration(duration time.Duration) string {
	if duration < time.Second {
		return "fast"
	} else if duration < 5*time.Second {
		return "normal"
	} else {
		return "slow"
	}
}
