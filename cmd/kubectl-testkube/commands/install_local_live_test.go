package commands

import (
	"testing"
	"time"
	"unicode/utf8"

	"github.com/pterm/pterm"
	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/localinstall"
)

func TestLiveRows_ReadWellAndNeverWrap(t *testing.T) {
	p := &localinstall.Progress{Ready: 2, Total: 10, Downloaded: 233_000_000, Services: []localinstall.ServiceProgress{
		{Role: "Background jobs", State: localinstall.StateWaiting, WaitsFor: "Messaging"},
		{Role: "Database", State: localinstall.StateReady, ReadyAfter: 41 * time.Second},
	}}

	rows := liveRows(p, 61*time.Second, 0, 200)

	assert.Equal(t, []string{
		"  ○ Background jobs   waiting for messaging",
		"  ✔ Database          ready in 41s",
		"  ⠋ 233 MB of ~1.2 GB · 2/10 ready · 1m 1s",
	}, stripColors(rows))
	for _, width := range []int{30, 60, 80} {
		for _, row := range stripColors(liveRows(p, 61*time.Second, 0, width)) {
			assert.LessOrEqual(t, utf8.RuneCountInString(row), width-1, "%q at width %d", row, width)
		}
	}
}

func stripColors(rows []string) []string {
	var out []string
	for _, r := range rows {
		out = append(out, pterm.RemoveColorFromString(r))
	}
	return out
}
