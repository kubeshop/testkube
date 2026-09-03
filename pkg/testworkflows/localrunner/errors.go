// Package localrunner implements the offline, developer-owned TestWorkflow loop
// behind `testkube local`. It deliberately never creates a Testkube API client.
package localrunner

import (
	"errors"
	"fmt"
)

const (
	// DefaultNamespace is deliberately separate from a normal Testkube installation.
	DefaultNamespace = "testkube-local"

	// LocalLabel marks resources that belong to the offline local-runner feature.
	LocalLabel = "testkube.io/local"
	// LocalRunIDLabel scopes a resource to one exact local invocation.
	LocalRunIDLabel = "testkube.io/local-run-id"
	// LocalComponentLabel differentiates auxiliary source relays from workflow resources.
	LocalComponentLabel = "testkube.io/local-component"

	localLabelValue = "true"
)

// CommandError is returned for a user-correctable command failure. The CLI root
// recognizes ExitCode so a malformed workflow never looks like a failed test.
type CommandError struct {
	code int
	err  error
}

func (e *CommandError) Error() string { return e.err.Error() }
func (e *CommandError) Unwrap() error { return e.err }
func (e *CommandError) ExitCode() int { return e.code }

// UsageError formats an error that must return POSIX-style command exit code 2.
func UsageError(format string, args ...any) error {
	return &CommandError{code: 2, err: fmt.Errorf(format, args...)}
}

// IsUsageError reports whether an error is expected to return exit code 2.
func IsUsageError(err error) bool {
	var commandError *CommandError
	return errors.As(err, &commandError) && commandError.code == 2
}

// ExecutionError marks a workflow or Kubernetes runtime failure. It remains a
// normal failure (exit status 1), rather than looking like invalid CLI input.
func ExecutionError(format string, args ...any) error {
	return &CommandError{code: 1, err: fmt.Errorf(format, args...)}
}

// InterruptedError preserves the conventional status for a developer who
// stops a local run with Ctrl-C. The caller still owns cleanup before returning
// this error.
func InterruptedError(err error) error {
	if err == nil {
		err = errors.New("local workflow run interrupted")
	}
	return &CommandError{code: 130, err: err}
}

// IsInterruptedError reports whether the local command should clean a run even
// when it was started with --keep. Interrupting a run is an explicit stop, not
// a request to preserve a possibly active workload.
func IsInterruptedError(err error) bool {
	var commandError *CommandError
	return errors.As(err, &commandError) && commandError.code == 130
}
