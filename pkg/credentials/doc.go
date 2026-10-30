// Package credentials resolves the values a Test Workflow asks the Control Plane for at runtime, through
// expression functions that the init process registers (cmd/testworkflow-init); the toolkit also registers
// credential(). The workflow processor leaves these calls unresolved, so they run inside the execution pod.
// identity_token() is registered by the init process only: the toolkit passes it on unresolved to the pods it
// spawns, so a token never ends up in a pod spec.
//
// # credential()
//
// credential(name[, computed]) returns a stored credential through the GetCredential RPC.
//
// # identity_token()
//
// identity_token(audience) returns a short-lived, signed identity token for the execution, with audience as its
// "aud" claim, through the GetIdentityToken RPC. A workflow exchanges it with a cloud provider that trusts the
// Control Plane as an identity provider (federated identity), for example as AWS_WEB_IDENTITY_TOKEN, instead of
// storing long-lived cloud credentials. Each call mints a new token and nothing is cached.
//
// A token in an environment variable is minted when the init process resolves the step's environment, before the
// command starts, and is not refreshed. A step that runs longer than the token's lifetime, or that is paused on
// start (paused: true) for longer than that, uses an expired token. Such steps need a refreshable token, for
// example written to a file that the cloud SDK reads again (AWS_WEB_IDENTITY_TOKEN_FILE), which this function does
// not provide.
//
// # Contract with the Control Plane
//
// The Control Plane (testkube-cloud-api) implements both RPCs. The standalone control plane (pkg/controlplane)
// returns Unimplemented for GetCredential, and implements GetIdentityToken when it has a signing key
// (IDENTITY_TOKEN_PRIVATE_KEY_FILE); without one it returns FailedPrecondition. Either failure fails the
// expression. Both sign identity tokens with pkg/identitytoken, which also serves the issuer documents.
//
// Resolved values are handed to observers, which the init process uses to mask them in the logs.
package credentials
