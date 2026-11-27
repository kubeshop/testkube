// Package controlplaneclient implements authenticated Control Plane clients.
//
// GetGitAccess forwards a repository URL and revision using the current execution
// identity. It does not accept tenant or execution overrides. The Control Plane's
// internal/agentserver package in testkube-cloud-api verifies that the execution
// belongs to an active GitLab Quality Loop binding before returning relay access.
//
// AgentTurnClient starts, reads and cancels one turn of an AI Agent with the current
// execution identity. It accepts no tenant or execution parameters. Each call retries
// a transient error, and the request ID of the start call makes that retry return the
// same turn.
package controlplaneclient
