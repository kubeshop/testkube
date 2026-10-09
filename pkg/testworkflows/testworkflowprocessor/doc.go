// Package testworkflowprocessor turns a test workflow into the Kubernetes resources of one
// execution, and deploys them.
//
// # Error messages
//
// An error of this package becomes the message of an execution that could not start. The start
// reason code names the part that failed, so the message names only the cause:
//
//   - When Kubernetes refuses the job, the message reads its structured causes, not its text. It
//     lists each cause one time, without the generated job ID. It also drops the container part of
//     the field path, because Kubernetes repeats a cause for each container of the pod. For
//     example: `the job is invalid: volumeMounts[4].name: Not found: "missing-volume"`.
//   - Without structured causes, the message keeps the text of Kubernetes.
package testworkflowprocessor
