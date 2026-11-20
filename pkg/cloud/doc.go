// Package cloud contains the generated Control Plane gRPC contract.
//
// GetGitAccess requests temporary GitLab relay credentials for an execution's
// repository URL and exact revision. The server derives organization, environment,
// Runner and execution identity from authentication metadata; request fields do
// not grant access. The corresponding authorization lives in testkube-cloud-api's
// internal/agentserver package. Deploy both ends before using authType: gitlab.
//
// StartAgentTurn, GetAgentTurn and CancelAgentTurn run one turn of an AI Agent for
// the agent step of a Test Workflow. The server takes the organization, the
// environment and the execution from the execution token, and no request field
// grants access. A start call with a known request ID returns the turn of the
// first call. A control plane that implements the three calls publishes the
// capability "tw-agent-step" (pkg/capabilities), and a caller checks it first.
package cloud
