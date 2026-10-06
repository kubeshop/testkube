// Package controller contains the reconcilers that sync Kubernetes resources to the Control Plane store.
//
// Some store rejections do not clear on retry: an ownership conflict or an invalid resource (see
// syncagent.IsRejection). terminalOnRejection wraps such an error in reconcile.TerminalError, so
// controller-runtime stops the requeue of the resource. The reconcilers return every other error
// unchanged, and controller-runtime retries it with backoff. The reconcilers do not log these errors,
// because controller-runtime logs the returned error with the kind and the name of the resource.
//
// skipRejectedResource in cmd/api-server/superagentmigration.go applies the same rule during the
// SuperAgent migration. Without it, the migration retries a rejected resource forever.
package controller
