// Package controller watches the job and the pod of an execution and builds the execution result from them.
//
// # Recorded cause
//
// While the pod waits, Kubernetes can report why the pod does not run. Kubernetes retries after these causes, so they do
// not stop the execution. The watcher builds a new state on each update, so the state computes the cause and does not
// store it.
//
// The notifier writes the cause and its reason code into the step that waits for its container. That is the
// initialization step until it finishes, then the first step that did not start. The notifier changes only a message
// that is empty or that it wrote. It keeps the cause after the execution completes, because Kubernetes deletes the pod.
// While the pod waits, the notifier sends the result without an alignment, so the execution stays scheduling.
//
// # Initialization timeout
//
// The initialization timeout counts from the job creation and ends when the first step container starts. When it ends,
// the watch asks one time for an abort and continues to read the execution. The abort sends no detail, because the
// initialization message already holds the cause. When a step container started, its start time decides and not the
// timer. A watch that connects late, for example to read the logs, finds a timer that already fired.
//
// # Completion
//
// The execution is complete when the pod finishes or when every step container terminates. Another container, for
// example a sidecar that a webhook adds, can keep the pod running after the last step. Without this rule, the watch
// does not end, and the execution stays running although every step has a result. The pod watcher keeps its own
// rule, because it must watch the pod until the runner deletes it.
package controller
