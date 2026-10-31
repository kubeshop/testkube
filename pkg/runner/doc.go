// Package runner starts Test Workflow executions on the agent and follows them to the end.
//
// # Cleanup retries
//
// After a workflow ends, destroyResources runs worker.Destroy (cleanup of the execution's
// Secrets and Pods) through the shared retry helper, bounded by CleanupResourcesRetryCount and
// CleanupResourcesRetryDelay, so a brief kube-apiserver blip during teardown does not leave orphan
// resources in the customer namespace.
package runner
