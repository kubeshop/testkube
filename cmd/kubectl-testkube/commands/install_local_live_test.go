package commands

import (
	"bytes"
	"strings"
	"sync/atomic"
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

func TestShowPlain_LogsStatusLinesWithoutTerminalCodes(t *testing.T) {
	was := plainEvery
	plainEvery = 20 * time.Millisecond
	t.Cleanup(func() { plainEvery = was })
	var progress atomic.Pointer[localinstall.Progress]
	progress.Store(&localinstall.Progress{Ready: 2, Total: 10, Downloaded: 233_000_000})
	var out bytes.Buffer

	stop := showPlain(&out, time.Now(), &progress)
	time.Sleep(70 * time.Millisecond)
	stop()

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	assert.Equal(t, "First run takes 3 to 8 minutes", strings.TrimSpace(lines[0]))
	assert.GreaterOrEqual(t, len(lines), 3)
	assert.Contains(t, lines[1], "Still installing · 233 MB of ~1.2 GB · 2/10 ready · ")
	assert.NotContains(t, out.String(), "\r", "no spinner frames in logs")
	assert.NotContains(t, out.String(), "\033[", "no cursor moves in logs")
}

func stripColors(rows []string) []string {
	var out []string
	for _, r := range rows {
		out = append(out, pterm.RemoveColorFromString(r))
	}
	return out
}
