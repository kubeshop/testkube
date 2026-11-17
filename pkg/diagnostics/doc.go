// Package diagnostics runs the validator groups behind `testkube diagnostics`, whose commands live
// in cmd/kubectl-testkube/commands/diagnostics.
//
// The groups are install (validators/deps), agent (validators/agent: the agent pods - api-server,
// testkube-api or testkube-runner - are ready) and license.
//
// To add a check, implement validators.Validator in a package under validators, register it in
// the group's Register*Validators function, and build the commands through New(cmd) so --output
// json applies.
//
// A validator that needs the cluster takes a connect function rather than a client, so an
// unreachable cluster is reported as a result (WithBreak) instead of aborting the run. Do not add
// checks for the deprecated operator.
package diagnostics
