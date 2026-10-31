// Package commands holds the root and top-level commands of the Testkube CLI.
//
// # On-prem demo install
//
// `testkube init demo` (init.go) installs the On-Prem demo on the new architecture: the Control
// Plane (the enterprise chart with values.demo.v2.yaml) plus a separate listener-enabled runner
// (kubeshop/testkube-runner). The bundled agent is gone. The legacy values.demo.yaml profile
// (bundled agent, MongoDB) is deprecated but kept for older CLIs.
//
// The CLI generates one agent secret key per install (common.GenerateDemoAgentSecretKey) and
// passes the same key to both sides: into the Control Plane's bootstrapConfig runner, so the
// Control Plane provisions it, and into the runner install
// (common.HelmUpgradeOrInstallTestkubeOnPremDemoRunner, then demoRunnerHelmOptions). No secret is
// baked into the binary or the chart. The runner identity (demoRunnerID, OrgID, EnvID) must stay
// in sync with the runner declared under bootstrapConfig in values.demo.v2.yaml, in
// testkube-cloud-charts.
//
// # Install lifecycle events
//
// `init demo` and `install local` report cli_install_started (before the Helm install; after the
// license step for install local) and cli_install_finished (after success) to the license
// service at POST https://license.testkube.io/events. The client lives in
// pkg/diagnostics/validators/license/client.go (Client.ReportEvent, EventRequest, and the
// LicenseEventsURL and EventCLIInstall* constants).
//
// The reportLicenseEvent and waitLicenseEvents helpers in init.go make delivery telemetry-gated
// (install local also skips them when its install tracker is off, for example under
// DO_NOT_TRACK), non-blocking (a background goroutine), and flushed before exit with a bounded
// wait. The license key in the body is the credential, validated by the worker before it records
// anything, so no shared secret ships in the CLI.
//
// A new lifecycle event needs an EventCLIInstall* constant, a reportLicenseEvent call, and an
// entry in the allowlist of the license worker's /events handler (testkube-infrastructure).
// `init demo` also sends init_demo_started and init_demo_finished to the PostHog install tracker,
// joined to install local runs by machine ID. Keep ARCHITECTURE.md in sync.
package commands
