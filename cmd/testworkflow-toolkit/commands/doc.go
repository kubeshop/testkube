// Package commands implements the Test Workflow toolkit commands.
//
// GitLab checkout uses authType: gitlab and an exact revision. The clone command
// obtains temporary relay credentials through controlplaneclient.GetGitAccess;
// the Control Plane validates the authenticated execution and repository binding.
// Relay credentials use a private temporary file and command-scoped helper options,
// preserving global Git CA/proxy settings without saving helpers in the checkout.
// Workflow definitions never need the GitLab OAuth token. A Control Plane without
// the RPC or feature enabled rejects checkout with an actionable error.
package commands
