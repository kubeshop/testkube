// Package telemetry builds and sends telemetry events, and identifies the cluster they come from.
//
// DO_NOT_TRACK=1 (any value except empty, 0 or false) skips every send, in the CLI and in the
// agent.
//
// # Senders
//
// Every Sender returns the body it transmitted (for Segment, the queued Track events as JSON),
// and sendData returns them one sender per line, so users can inspect telemetry before
// consenting: the CLI prints it as "telemetry send event response" under --verbose. A new sender
// must return what it sends, not a response status alone.
//
// # Cluster type detection
//
// cluster_type.go detects the Kubernetes cluster type in layers: node providerID, then node
// labels, then the server version, then kube-system pod names. The result is cached with
// sync.Once. To support a new cluster type, add detection entries to the right layers in
// cluster_type.go and test cases in cluster_type_test.go.
//
// # Agent heartbeat
//
// cmd/api-server/services/telemetry.go drives the loop that sends testkube_api_heartbeat events
// hourly, with the detected cluster type and the agent's capabilities.
// cmd/api-server/services/capabilities.go extracts those capability tags (persona, mode, feature
// flags) from the runtime config; a new agent feature or toggle that should be tracked goes there
// and in capabilities_test.go.
//
// The hosted-runner tag marks runners that Testkube provisions for trial users, detected from the
// tkcagent_hr_ prefix the Control Plane assigns to RUNNER_NAME. Keep hostedRunnerNamePrefix in
// sync with naming.HostedRunnerAgentName in testkube-cloud-api.
//
// # CLI payloads
//
// The CLI runtime context, including the AI tool that invoked the CLI, comes from the cliruntime
// package. Telemetry surfaces the AI tool as the ai_tool param (Segment property aiTool) across
// all CLI-origin payloads: NewCLIPayload, NewCLIWithLicensePayload, the inline error and preview
// payloads in telemetry.go, and the MCP tool payload in mcp.go.
package telemetry
