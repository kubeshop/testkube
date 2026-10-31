// Package data holds the state of the init process and the expressions that read it.
//
// The init process keeps one state for the pod. The state holds the steps, their results, and the values that the
// steps publish. It writes the state to a file, so each step container of the pod reads what the steps before it did.
//
// # Step expressions
//
// A workflow reads the state of an earlier step through the step machine. The machine resolves these names:
//
//   - step.results, the results directory of the current step
//   - step.<id>.results, the results directory of an earlier step
//   - step.<id>.outputs.<key>, a value that an earlier step published
//   - step.<id>.status, the status of an earlier step
//   - step.<id>.exitCode, the exit code of an earlier step
//
// The <id> is the stable id of the step, and not the reference that Testkube generates for the execution.
//
// An accessor that has no value leaves the expression unresolved. A step that did not end has no status. A step that
// a condition skipped has the status skipped but no exit code, because it ran no command. Without this rule the
// skipped step reports the exit code 0, which reads as a success.
package data
