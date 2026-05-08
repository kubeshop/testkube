package telemetry

import (
	"runtime"
	"time"

	"github.com/kubeshop/testkube/pkg/utils/text"
)

// SendMCPToolEventWithContext sends telemetry for MCP tool execution with explicit context
func SendMCPToolEventWithContext(toolName string, duration time.Duration, hasError bool, version string, runContext RunContext, source string) (string, error) {
	payload := NewMCPToolPayloadWithContext(toolName, "testkube_mcp_tool_execution", duration, version, hasError, runContext, source)
	return sendData(senders, payload)
}

// NewMCPToolPayloadWithContext creates a payload for MCP tool telemetry events with custom context
func NewMCPToolPayloadWithContext(toolName, eventName string, duration time.Duration, version string, hasError bool, runContext RunContext, source string) Payload {
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
				Context:         runContext, // Use the provided context instead of getCurrentContext()
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
				Source:     source,
			},
		}},
	}
}
