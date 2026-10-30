// Package cloud contains the generated Control Plane gRPC contract.
//
// GetGitAccess requests temporary GitLab relay credentials for an execution's
// repository URL and exact revision. The server derives organization, environment,
// Runner and execution identity from authentication metadata; request fields do
// not grant access. The corresponding authorization lives in testkube-cloud-api's
// internal/agentserver package. Deploy both ends before using authType: gitlab.
package cloud
