// Package common holds what the CLI commands share: the client, flags, Helm options, input
// helpers and the update check.
//
// # Update check
//
// update_check.go implements MaybeNotifyNewerRelease (a hint after each command) and
// CheckComponentsStatus (the per-component report `testkube version` renders). Both consult
// pkg/cliruntime to skip in CI, Docker and Kubernetes contexts, and honor the --output flag and
// the TESTKUBE_DISABLE_UPDATE_CHECK=1 opt-out, which suppresses both. The CLI persists
// lastUpdateCheckAt and latestKnownVersion in ~/.testkube/config.json to throttle the hint to once
// per day.
//
// install_source.go classifies how the running binary was installed (Homebrew, Chocolatey, APT,
// install.sh, Docker, go install, unknown) by inspecting the resolved os.Executable path and the
// Docker context; the classification picks the upgrade command the hint shows. A new install
// channel extends DetectInstallSource and adds a case to install_source_test.go that exercises
// the new path under the relevant goos. A new CI, runtime or AI-tool detection belongs in
// pkg/cliruntime, so telemetry and the update check stay in sync.
//
// # Login without a terminal
//
// GetClient (client.go) uses ui.StdinIsInteractive to refuse starting a login when a token
// refresh fails with no terminal present, the same way the email-link branch beside it does.
//
// # File input
//
// A flag that names a file to read takes - for the command's standard input (ReadFileOrStdin in
// input.go), so content can be piped: `cat payload.json | testkube create webhook
// --payload-template -`. Read new file flags through it, and use cmd.InOrStdin rather than
// os.Stdin so tests can supply the input.
//
// # Upgrading with --reuse-values
//
// `testkube upgrade --reuse-values` passes Helm's --reuse-values. Helm applies --set over reused
// values, so in that mode prepareCommonHelmArgs sends only the settings of flags the user passed
// (HelmOptions.ExplicitFlags, sendsFlag and sendsAnyFlag in helper.go): a flag default such as
// --no-postgres=true would otherwise silently turn off what the release had enabled. A new
// flag-derived Helm setting must go through sendsFlag.
//
// Never gate on a value being non-empty: ProcessMasterFlags composes a runner's control plane
// addresses from default prefixes and the root domain, so they always are, and sending them would
// point a runner of a custom control plane at SaaS. The flag lists (agentURIFlags, uiURIFlags)
// must name every flag ProcessMasterFlags reads for an address, the --*-uri-override flags
// included, or that flag is silently ignored. A runner's disabled MinIO, MongoDB and PostgreSQL
// are always sent.
package common
