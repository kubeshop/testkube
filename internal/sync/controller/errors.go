package controller

import (
	"context"
	"errors"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	syncagent "github.com/kubeshop/testkube/internal/sync"
)

// ownershipConflictReason is the Event reason recorded against a resource that the Control Plane
// refuses to sync because a different GitOps agent owns it.
const ownershipConflictReason = "GitOpsOwnershipConflict"

// reportSyncError decides how a failed store operation is reported to the user.
//
// An ownership conflict stands until somebody changes the resource or its owner, so retrying only
// produces noise. Those are recorded as a Warning on the resource and marked terminal to stop the
// requeue loop; controller-runtime still logs them and counts them in its metrics. Every other
// failure is returned unchanged so that it is retried with backoff.
//
// regarding is the resource the Event is attached to, and is nil when the resource has already been
// removed from the cluster and there is nothing left to annotate. The Control Plane's own message
// travels along in err, so it does not need to be repeated here.
func reportSyncError(ctx context.Context, recorder events.EventRecorder, regarding runtime.Object, err error) error {
	if !errors.Is(err, syncagent.ErrOwnershipConflict) {
		return err
	}

	ctrl.LoggerFrom(ctx).Error(err, "GitOps sync rejected by the Control Plane; not retrying")

	if recorder != nil && regarding != nil {
		recorder.Eventf(regarding, nil, corev1.EventTypeWarning, ownershipConflictReason, "Sync", "%s", err.Error())
	}

	return reconcile.TerminalError(err)
}
