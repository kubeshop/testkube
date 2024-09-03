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

func (n *notifier) Result(result testkube.TestWorkflowResult) {
	n.resultMu.Lock()
	n.result = result
	n.resultMu.Unlock()
	// TODO: Consider checking for change?
	n.scheduleFlush()
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

// TODO: Is it needed? Maybe should work differently?
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
	if !n.flushScheduled {
		n.flushMu.Unlock()
		return
	}
	n.resultMu.RLock()
	notification := Notification{Timestamp: n.result.LatestTimestamp(), Result: n.result.Clone()}
	n.resultMu.RUnlock()
	n.flushScheduled = false
	n.flushMu.Unlock()
	n.send(notification)
}

func (n *notifier) scheduleFlush() {
	n.flushMu.Lock()
	defer func() {
		recover() // ignore writing to closed channel
		n.flushMu.Unlock()
	}()

	// Inform scheduler about the next result
	n.flushScheduled = true
	select {
	case n.flushCh <- struct{}{}:
	default:
	}
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

func (n *notifier) Output(ref string, ts time.Time, output *instructions.Instruction) {
	if ref == InitContainerName {
		ref = ""
	} else if ref != "" {
		n.resultMu.RLock()
		if _, ok := n.result.Steps[ref]; !ok {
			n.resultMu.RUnlock()
			return
		}
		n.resultMu.RUnlock()
	}
	n.RegisterTimestamp(ref, ts)
	n.Flush()
	n.send(Notification{Timestamp: ts.UTC(), Ref: ref, Output: output})
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

	n := &notifier{
		ch:          make(chan ChannelMessage[Notification]),
		ctx:         ctx,
		sig:         sig,
		scheduledAt: scheduledAt,
		result:      result,
		lastTs:      make(map[string]time.Time),

		flushCh: make(chan struct{}, 1),
	}

	go func() {
		<-ctx.Done()
		close(n.ch)
	}()

	go func() {
		defer func() {
			close(n.flushCh)
		}()
		for {
			// Prioritize final flush when it is done
			select {
			case <-n.ctx.Done():
				n.Flush()
				return
			default:
			}

			// Wait until first message
			select {
			case <-n.ctx.Done():
				n.Flush()
				return
			case <-n.flushCh:
				maxTimer := time.NewTimer(FlushResultMaxTime)
			buffering:
				for {
					select {
					case <-n.ctx.Done():
						break buffering
					case <-maxTimer.C:
						n.Flush()
						break buffering
					case <-time.After(FlushResultTime):
						n.Flush()
						break buffering
					case <-n.flushCh:
						continue
					}
				}
				if !maxTimer.Stop() {
					<-maxTimer.C
				}
			}
		}
	}()

	return n
}
