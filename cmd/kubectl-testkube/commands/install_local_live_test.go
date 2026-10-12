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
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/localinstall"
)

func TestLiveRows_ReadWellAndNeverWrap(t *testing.T) {
	p := &localinstall.Progress{Ready: 2, Total: 10, Downloaded: 233_000_000, Services: []localinstall.ServiceProgress{
		{Role: "Background jobs", State: localinstall.StateWaiting, WaitsFor: "Messaging"},
		{Role: "Database", State: localinstall.StateReady, ReadyAfter: 41 * time.Second},
		{Role: "Runner", State: localinstall.StateWaiting, WaitsFor: "API"},
	}}

	rows := liveRows(p, 61*time.Second, 0, 200, 50)

	assert.Equal(t, []string{
		"  ○ Background jobs   waiting for messaging",
		"  ✔ Database          ready in 41s",
		"  ○ Runner            waiting for API",
		"  ⠋ 233 MB of ~1.2 GB · 2/10 ready · 1m 1s",
	}, stripColors(rows))
	for _, width := range []int{30, 60, 80} {
		for _, row := range stripColors(liveRows(p, 61*time.Second, 0, width, 50)) {
			assert.LessOrEqual(t, utf8.RuneCountInString(row), width-1, "%q at width %d", row, width)
		}
	}
}

func TestLiveRows_ShortWindowGetsOnlyTheFooter(t *testing.T) {
	p := &localinstall.Progress{Ready: 2, Total: 10, Services: make([]localinstall.ServiceProgress, 10)}

	assert.Len(t, liveRows(p, time.Second, 0, 80, 12), 11, "list plus cursor line fits")
	assert.Equal(t, []string{"  ⠋ 2/10 ready · 1s"}, stripColors(liveRows(p, time.Second, 0, 80, 11)))
	assert.Len(t, liveRows(p, time.Second, 0, 80, 0), 11, "unknown height keeps the list")
}

func TestShowPlain_LogsStatusLinesWithoutTerminalCodes(t *testing.T) {
	was := plainEvery
	plainEvery = 5 * time.Millisecond
	t.Cleanup(func() { plainEvery = was })
	var progress atomic.Pointer[localinstall.Progress]
	progress.Store(&localinstall.Progress{Ready: 2, Total: 10, Downloaded: 233_000_000})
	var out bytes.Buffer

	stop := showPlain(&out, time.Now(), &progress)
	time.Sleep(100 * time.Millisecond)
	stop()

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	assert.Equal(t, "First run takes 3 to 8 minutes", strings.TrimSpace(lines[0]))
	require.GreaterOrEqual(t, len(lines), 2)
	assert.Contains(t, lines[1], "Still installing · 233 MB of ~1.2 GB · 2/10 ready · ")
	assert.NotContains(t, out.String(), "\r", "no spinner frames in logs")
	assert.NotContains(t, out.String(), "\033[", "no cursor moves in logs")
}

func TestLiveArea_MovesUpExactlyTheRowsItDrew(t *testing.T) {
	var out bytes.Buffer
	a := &liveArea{out: &out}
	a.draw([]string{"hint"})
	out.Reset()

	a.draw([]string{"a", "b", "c"})
	assert.Equal(t, "\033[1A\r\033[Ja\nb\nc\n", out.String())
	out.Reset()

	a.clear()
	assert.Equal(t, "\033[3A\r\033[J", out.String())
}

func stripColors(rows []string) []string {
	var out []string
	for _, r := range rows {
		out = append(out, pterm.RemoveColorFromString(r))
	}
	return out
}
