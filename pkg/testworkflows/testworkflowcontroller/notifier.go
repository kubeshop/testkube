package testworkflowcontroller

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/instructions"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/stage"
	"github.com/kubeshop/testkube/pkg/ui"
)

const (
	FlushResultTime    = 50 * time.Millisecond
	FlushResultMaxTime = 100 * time.Millisecond
)

type notifier struct {
	ctx         context.Context
	ch          chan ChannelMessage[Notification]
	result      testkube.TestWorkflowResult
	sig         []testkube.TestWorkflowSignature
	scheduledAt time.Time
	lastTs      map[string]time.Time

	resultMu       sync.RWMutex
	flushMu        sync.Mutex
	flushCh        chan struct{}
	flushScheduled bool
}

func (n *notifier) send(value Notification) {
	// Ignore when the channel is already closed
	defer func() {
		recover()
	}()
	n.ch <- ChannelMessage[Notification]{Value: value}
}

func (n *notifier) error(err error) {
	// Ignore when the channel is already closed
	defer func() {
		recover()
	}()
	n.ch <- ChannelMessage[Notification]{Error: err}
}

func (n *notifier) unsafeGetLastTimestamp(ref string) time.Time {
	last := n.lastTs[ref]
	if n.result.Steps[ref].FinishedAt.After(last) {
		return n.result.Steps[ref].FinishedAt
	}
	if n.result.Steps[ref].StartedAt.After(last) {
		return n.result.Steps[ref].StartedAt
	}
	if n.result.Steps[ref].QueuedAt.After(last) {
		return n.result.Steps[ref].QueuedAt
	}
	return last
}

func (n *notifier) GetLastTimestamp(ref string) time.Time {
	n.resultMu.RLock()
	defer n.resultMu.RUnlock()
	return n.unsafeGetLastTimestamp(ref)
}

func (n *notifier) RegisterTimestamp(ref string, t time.Time) {
	if t.After(n.GetLastTimestamp(ref)) {
		n.resultMu.Lock()
		n.lastTs[ref] = t.UTC()
		n.resultMu.Unlock()
	}
}

func (n *notifier) Flush() {
	n.flushMu.Lock()
	defer n.flushMu.Unlock()
	if !n.flushScheduled {
		return
	}
	n.resultMu.RLock()
	defer n.resultMu.RUnlock()
	n.send(Notification{Timestamp: n.result.LatestTimestamp(), Result: n.result.Clone()})
	n.flushScheduled = false
}

func (n *notifier) scheduleFlush() {
	n.flushMu.Lock()
	defer n.flushMu.Unlock()

	// Inform existing scheduler about the next result
	if n.flushScheduled {
		select {
		case n.flushCh <- struct{}{}:
		default:
		}
		return
	}

	// Run the scheduler
	n.flushScheduled = true
	go func() {
		flushTimer := time.NewTimer(FlushResultMaxTime)
		flushTimerEnabled := false

		for {
			if n.ctx.Err() != nil {
				return
			}

			select {
			case <-n.ctx.Done():
				n.Flush()
				return
			case <-flushTimer.C:
				n.Flush()
				flushTimerEnabled = false
			case <-time.After(FlushResultTime):
				n.Flush()
				flushTimerEnabled = false
			case <-n.flushCh:
				if !flushTimerEnabled {
					flushTimerEnabled = true
					flushTimer.Reset(FlushResultMaxTime)
				}
				continue
			}
		}
	}()
}

func (n *notifier) Raw(ref string, ts time.Time, message string, temporary bool) {
	if message != "" {
		if ref == InitContainerName {
			ref = ""
		}
		// TODO: use timestamp from the message too for lastTs?
		n.Flush()
		n.send(Notification{
			Timestamp: ts.UTC(),
			Log:       message,
			Ref:       ref,
			Temporary: temporary,
		})
	}
}

func (n *notifier) Log(ref string, ts time.Time, message string) {
	if message != "" {
		n.RegisterTimestamp(ref, ts)
		n.Raw(ref, ts, fmt.Sprintf("%s %s", ts.Format(KubernetesLogTimeFormat), message), false)
	}
}

func (n *notifier) Error(err error) {
	n.error(err)
}

func (n *notifier) Event(ref string, ts time.Time, level, reason, message string) {
	n.RegisterTimestamp(ref, ts)
	color := ui.LightGray
	if level != "Normal" {
		color = ui.Yellow
	}
	log := color(fmt.Sprintf("(%s) %s", reason, message))
	n.Raw(ref, ts, fmt.Sprintf("%s %s\n", ts.Format(KubernetesLogTimeFormat), log), level == "Normal")
}

func (n *notifier) recompute() {
	lastTs := n.unsafeGetLastTimestamp("")
	if !n.result.Initialization.FinishedAt.IsZero() && n.result.Initialization.FinishedAt.Before(lastTs) {
		n.result.Initialization.FinishedAt = lastTs
	}
	for k := range n.result.Steps {
		if !n.result.Steps[k].FinishedAt.IsZero() && n.result.Steps[k].FinishedAt.Before(lastTs) {
			step := n.result.Steps[k]
			step.FinishedAt = lastTs
			n.result.Steps[k] = step
		}
	}
	n.result.Recompute(n.sig, n.scheduledAt)
}

func (n *notifier) emit() {
	n.recompute()
	n.scheduleFlush()
}

func (n *notifier) queue(ts time.Time) {
	if n.result.QueuedAt.Equal(ts) {
		return
	}
	n.result.QueuedAt = ts.UTC()
	n.emit()
}

func (n *notifier) queueInit(ts time.Time) {
	if n.result.Initialization.QueuedAt.Equal(ts) {
		return
	}
	n.result.Initialization.QueuedAt = ts.UTC()
	n.emit()
}

func (n *notifier) queueStep(ref string, ts time.Time) {
	if n.result.Steps[ref].QueuedAt.Equal(ts) {
		return
	}
	s := n.result.Steps[ref]
	s.QueuedAt = ts.UTC()
	n.result.Steps[ref] = s
	n.emit()
}

func (n *notifier) Queue(ref string, ts time.Time) {
	n.resultMu.Lock()
	defer n.resultMu.Unlock()
	if ref == "" {
		n.queue(ts)
	} else if ref == InitContainerName {
		n.queueInit(ts)
	} else {
		n.queueStep(ref, ts)
	}
}

func (n *notifier) start(ts time.Time) {
	if n.result.StartedAt.Equal(ts) {
		return
	}
	n.result.StartedAt = ts.UTC()
	if n.result.Status == nil || *n.result.Status == testkube.QUEUED_TestWorkflowStatus {
		n.result.Status = common.Ptr(testkube.RUNNING_TestWorkflowStatus)
	}
	n.emit()
}

func (n *notifier) startInit(ts time.Time) {
	if n.result.Initialization.StartedAt.Equal(ts) {
		return
	}
	n.result.Initialization.StartedAt = ts.UTC()
	if n.result.Initialization.Status == nil || *n.result.Initialization.Status == testkube.QUEUED_TestWorkflowStepStatus {
		n.result.Initialization.Status = common.Ptr(testkube.RUNNING_TestWorkflowStepStatus)
	}
	n.emit()
}

func (n *notifier) startStep(ref string, ts time.Time) {
	if n.result.Steps[ref].StartedAt.Equal(ts) {
		return
	}
	s := n.result.Steps[ref]
	s.StartedAt = ts.UTC()
	if s.Status == nil || *s.Status == testkube.QUEUED_TestWorkflowStepStatus {
		s.Status = common.Ptr(testkube.RUNNING_TestWorkflowStepStatus)
	}
	n.result.Steps[ref] = s
	n.emit()
}

func (n *notifier) Start(ref string, ts time.Time) {
	n.resultMu.Lock()
	defer n.resultMu.Unlock()

	if ref == "" {
		n.start(ts)
	} else if ref == InitContainerName {
		n.startInit(ts)
	} else {
		n.startStep(ref, ts)
	}
}

func (n *notifier) Output(ref string, ts time.Time, output *instructions.Instruction) {
	n.resultMu.RLock()
	if ref == InitContainerName {
		ref = ""
	}
	if _, ok := n.result.Steps[ref]; !ok && ref != "" {
		n.resultMu.RUnlock()
		return
	}
	n.resultMu.RUnlock()
	n.RegisterTimestamp(ref, ts)
	n.Flush()
	n.send(Notification{Timestamp: ts.UTC(), Ref: ref, Output: output})
}

func (n *notifier) Finish(ts time.Time) {
	if ts.IsZero() {
		return
	}
	n.resultMu.Lock()
	defer n.resultMu.Unlock()
	n.result.FinishedAt = ts
	n.emit()
}

func (n *notifier) UpdateStepStatus(ref string, status testkube.TestWorkflowStepStatus) {
	n.resultMu.Lock()
	defer n.resultMu.Unlock()
	if _, ok := n.result.Steps[ref]; !ok || (n.result.Steps[ref].Status != nil || *n.result.Steps[ref].Status == status) {
		return
	}
	n.result.UpdateStepResult(n.sig, ref, testkube.TestWorkflowStepResult{Status: &status}, n.scheduledAt)
	n.emit()
}

func (n *notifier) finishInit(status ContainerResultStep) {
	if n.result.Initialization.FinishedAt.Equal(status.FinishedAt) && n.result.Initialization.Status != nil && *n.result.Initialization.Status == status.Status && (status.Status != testkube.ABORTED_TestWorkflowStepStatus || n.result.Initialization.ErrorMessage == status.Details) {
		return
	}
	n.result.Initialization.FinishedAt = status.FinishedAt.UTC()
	n.result.Initialization.Status = common.Ptr(status.Status)
	n.result.Initialization.ExitCode = float64(status.ExitCode)
	n.result.Initialization.ErrorMessage = status.Details
	n.emit()
}

func (n *notifier) IsAnyAborted() bool {
	n.resultMu.RLock()
	defer n.resultMu.RUnlock()
	if n.result.Initialization.Status != nil && *n.result.Initialization.Status == testkube.ABORTED_TestWorkflowStepStatus {
		return true
	}
	for _, s := range n.result.Steps {
		if s.Status != nil && *s.Status == testkube.ABORTED_TestWorkflowStepStatus {
			return true
		}
	}
	return false
}

func (n *notifier) IsFinished(ref string) bool {
	n.resultMu.RLock()
	defer n.resultMu.RUnlock()
	if ref == InitContainerName {
		return !n.result.Initialization.FinishedAt.IsZero()
	}
	return !n.result.Steps[ref].FinishedAt.IsZero()
}

func (n *notifier) FinishStep(ref string, status ContainerResultStep) {
	n.resultMu.Lock()
	defer n.resultMu.Unlock()
	if ref == InitContainerName {
		n.finishInit(status)
		return
	}
	if n.result.Steps[ref].FinishedAt.Equal(status.FinishedAt) && n.result.Steps[ref].Status != nil && *n.result.Steps[ref].Status == status.Status && (status.Status != testkube.ABORTED_TestWorkflowStepStatus || n.result.Steps[ref].ErrorMessage == status.Details) {
		return
	}
	s := n.result.Steps[ref]
	s.FinishedAt = status.FinishedAt.UTC()
	s.Status = common.Ptr(status.Status)
	s.ExitCode = float64(status.ExitCode)
	s.ErrorMessage = status.Details
	n.result.Steps[ref] = s
	n.emit()
}

func (n *notifier) Pause(ref string, ts time.Time) {
	n.resultMu.Lock()
	defer n.resultMu.Unlock()
	if n.result.Steps[ref].Status != nil && *n.result.Steps[ref].Status == testkube.PAUSED_TestWorkflowStepStatus {
		return
	}
	n.result.PauseStart(n.sig, n.scheduledAt, ref, ts)
	n.emit()
}

func (n *notifier) Resume(ref string, ts time.Time) {
	n.resultMu.Lock()
	defer n.resultMu.Unlock()
	n.result.PauseEnd(n.sig, n.scheduledAt, ref, ts)
	n.emit()
}

func (n *notifier) GetStepResult(ref string) testkube.TestWorkflowStepResult {
	n.resultMu.RLock()
	defer n.resultMu.RUnlock()
	if ref == InitContainerName {
		return *n.result.Initialization
	}
	return n.result.Steps[ref]
}

func newNotifier(ctx context.Context, signature []stage.Signature, scheduledAt time.Time) *notifier {
	// Initialize the zero result
	sig := make([]testkube.TestWorkflowSignature, len(signature))
	for i, s := range signature {
		sig[i] = s.ToInternal()
	}
	result := testkube.TestWorkflowResult{
		Status:          common.Ptr(testkube.QUEUED_TestWorkflowStatus),
		PredictedStatus: common.Ptr(testkube.PASSED_TestWorkflowStatus),
		Initialization: &testkube.TestWorkflowStepResult{
			Status: common.Ptr(testkube.QUEUED_TestWorkflowStepStatus),
		},
		Steps: stage.MapSignatureListToStepResults(signature),
	}
	result.Recompute(sig, scheduledAt)

	ch := make(chan ChannelMessage[Notification])

	go func() {
		<-ctx.Done()
		close(ch)
	}()

	return &notifier{
		ch:          ch,
		ctx:         ctx,
		sig:         sig,
		scheduledAt: scheduledAt,
		result:      result,
		lastTs:      make(map[string]time.Time),

		flushCh: make(chan struct{}, 1),
	}
}
