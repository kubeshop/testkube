// Package controller contains the reconcilers that sync Kubernetes resources to the Control Plane store.
//
// Each reconciler works out the syncagent.Target of a resource from the namespace of the request,
// with the syncagent.TargetFunc that main.go builds from GITOPS_NAMESPACES, and passes it to the
// store on every update and delete.
//
// Two watched namespaces that sync into the same Environment can hold a resource of the same kind
// and name, and the Control Plane keys a resource by its name. Before an update, checkNameCollision
// lists the kind from the cache and refuses the resource with syncagent.ErrNameCollision when
// another one wins. The resource with the earlier creationTimestamp wins, and on a tie the
// namespace that sorts first. The rule depends only on the objects, so every replica comes to the
// same answer. The refusal is terminal. A List error stays retryable.
//
// A delete must not remove the record of another namespace. When another watched namespace of the
// same Environment still holds a resource of the same name, syncDelete sends the winning one of
// them instead of the delete. The delete of a refused resource so keeps the winner, and the delete
// of the winner hands the record to the next one. With gitOpsOwnership on, the Control Plane refuses
// that update with an ownership conflict when the record belongs to the deleted resource. On such a
// conflict, or on any other rejection of that update, such as an invalid resource, syncDelete
// deletes the record, and the next resource syncs after its next change.
//
// Some store rejections do not clear on retry: an ownership conflict, an invalid resource, a
// forbidden Environment or a name collision (see syncagent.IsRejection). terminalOnRejection wraps
// such an error in reconcile.TerminalError, so controller-runtime stops the requeue of the resource. The reconcilers return every other error
// unchanged, and controller-runtime retries it with backoff. The reconcilers do not log these errors,
// because controller-runtime logs the returned error with the kind and the name of the resource.
//
// skipRejectedResource in cmd/api-server/superagentmigration.go applies the same rule during the
// SuperAgent migration. Without it, the migration retries a rejected resource forever.
package controller
