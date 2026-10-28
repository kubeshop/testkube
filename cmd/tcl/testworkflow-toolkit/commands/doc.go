// Package commands holds the commercial testworkflow-toolkit commands.
//
// # Services cleanup retries
//
// kill.go runs the services cleanup of the kill step - worker.List, each getServiceHealth pod
// read, and worker.DestroyGroup - through the retryTransient helper and the classifiers in
// retry.go, bounded by cleanupRetryCount and cleanupRetryDelay. Only a failure whose every branch
// is transient is downgraded to a warning: the test steps already have their result, so a
// teardown blip is not the test's verdict. Anything else still fails the step. retry.go documents
// which error classes count as transient and why.
package commands
