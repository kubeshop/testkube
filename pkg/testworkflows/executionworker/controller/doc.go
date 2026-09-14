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
package controller
