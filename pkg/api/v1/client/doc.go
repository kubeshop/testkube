// Package client implements the agent API clients the CLI uses, both straight to
// an agent and through the Control Plane (--client=cloud).
//
// Through the Control Plane, agent requests go to
// /organizations/{org}/environments/{env}/agent/v1/<resource> (see
// CloudAgentPathPrefix). The Control Plane's agent proxy in testkube-cloud-api
// (stripAgentAPIVersion in internal/server/http.go) accepts the stated version and
// forwards /v1/<resource> to the agent. Releases before 2.14.0 add /v1 themselves,
// so this CLI needs a Control Plane on 2.14.0 or later.
//
// Artifact downloads ask the Control Plane's own endpoint, outside the agent proxy,
// for a signed URL, then download from that URL.
package client
