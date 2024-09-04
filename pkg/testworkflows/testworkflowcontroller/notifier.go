package testworkflowcontroller

import (
	"context"
	"fmt"
	"time"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/instructions"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/stage"
	"github.com/kubeshop/testkube/pkg/ui"
)

type notifier struct {
	ctx  context.Context
	ch   chan ChannelMessage[Notification]
	refs map[string]struct{}
}

func (n *notifier) send(value Notification) {
	// Ignore when the channel is already closed
	defer func() {
		recover()
	}()
	select {
	case <-n.ctx.Done():
	case n.ch <- ChannelMessage[Notification]{Value: value}:
	}
}

func (n *notifier) error(err error) {
	// Ignore when the channel is already closed
	defer func() {
		recover()
	}()
	select {
	case <-n.ctx.Done():
	case n.ch <- ChannelMessage[Notification]{Error: err}:
	}
}

func (n *notifier) Result(result testkube.TestWorkflowResult) {
	// TODO: Find a way to avoid sending if it is identical
	n.send(Notification{Timestamp: result.LatestTimestamp(), Result: &result})
}

func (n *notifier) Raw(ref string, ts time.Time, message string, temporary bool) {
	if message != "" {
		if ref == InitContainerName {
			ref = ""
		}
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
		n.Raw(ref, ts, fmt.Sprintf("%s %s", ts.Format(KubernetesLogTimeFormat), message), false)
	}
}

func (n *notifier) Error(err error) {
	n.error(err)
}

func (n *notifier) Event(ref string, ts time.Time, level, reason, message string) {
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
		if _, ok := n.refs[ref]; !ok {
			return
		}
	}
	n.send(Notification{Timestamp: ts.UTC(), Ref: ref, Output: output})
}

func newNotifier(ctx context.Context, signature []stage.Signature) *notifier {
	refs := make(map[string]struct{})
	for _, s := range stage.MapSignatureToSequence(signature) {
		refs[s.Ref()] = struct{}{}
	}

	return &notifier{
		ch:   make(chan ChannelMessage[Notification]),
		ctx:  ctx,
		refs: refs,
	}
}
