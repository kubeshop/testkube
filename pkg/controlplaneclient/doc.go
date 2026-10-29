// Package controlplaneclient implements authenticated Control Plane clients.
//
// GetGitAccess forwards a repository URL and revision using the current execution
// identity. It does not accept tenant or execution overrides. The Control Plane's
// internal/agentserver package in testkube-cloud-api verifies that the execution
// belongs to an active GitLab Quality Loop binding before returning relay access.
package controlplaneclient
