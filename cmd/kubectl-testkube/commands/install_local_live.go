package commands

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/mattn/go-isatty"
	"github.com/pterm/pterm"

	"github.com/kubeshop/testkube/pkg/localinstall"
	"github.com/kubeshop/testkube/pkg/ui"
)

var liveFrames = []string{"⠋", "⠹", "⠼", "⠦", "⠇"}

// Fits the widest role, "Background jobs", plus a gap.
const roleWidth = 18

func stdoutIsTerminal() bool {
	fd := os.Stdout.Fd()
	return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
}

// pterm's printers race on text and break on wrapped lines.
type liveArea struct {
	out   io.Writer
	lines int
}

// One write per frame, so the area never tears.
func (a *liveArea) draw(rows []string) {
	var b strings.Builder
	if a.lines > 0 {
		fmt.Fprintf(&b, "\033[%dA", a.lines)
	}
	b.WriteString("\r\033[J")
	for _, r := range rows {
		b.WriteString(r + "\n")
	}
	a.lines = len(rows)
	_, _ = io.WriteString(a.out, b.String())
}

func (a *liveArea) clear() {
	if a.lines > 0 {
		_, _ = fmt.Fprintf(a.out, "\033[%dA\r\033[J", a.lines)
	}
	a.lines = 0
}

func showLive(out io.Writer, start time.Time, progress *atomic.Pointer[localinstall.Progress]) (stop func()) {
	area := &liveArea{out: out}
	done, finished := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(finished)
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()
		for frame := 0; ; frame++ {
			width, height, err := pterm.GetTerminalSize()
			if err != nil {
				width, height = pterm.GetTerminalWidth(), 0
			}
			area.draw(liveRows(progress.Load(), time.Since(start), frame, width, height))
			select {
			case <-done:
				area.clear()
				return
			case <-ticker.C:
			}
		}
	}()
	return func() {
		close(done)
		<-finished
	}
}

// Logs and CI keep every line, so print rarely.
var plainEvery = 30 * time.Second

func showPlain(out io.Writer, start time.Time, progress *atomic.Pointer[localinstall.Progress]) (stop func()) {
	_, _ = fmt.Fprintln(out, "  First run takes 3 to 8 minutes")
	done, finished := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(finished)
		ticker := time.NewTicker(plainEvery)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				_, _ = fmt.Fprintln(out, "  "+plainStatus(progress.Load(), time.Since(start)))
			}
		}
	}()
	return func() {
		close(done)
		<-finished
	}
}

func plainStatus(p *localinstall.Progress, elapsed time.Duration) string {
	if p == nil {
		return "Still installing · " + took(elapsed)
	}
	return "Still installing · " + downloadedText(p.Downloaded) + fmt.Sprintf("%d/%d ready · ", p.Ready, p.Total) + took(elapsed)
}

// Cut before coloring: a wrapped row breaks the redraw.
func liveRows(p *localinstall.Progress, elapsed time.Duration, frame, width, height int) []string {
	spin := ui.LightCyan(liveFrames[frame%len(liveFrames)])
	if p == nil {
		return []string{"  " + spin + " " + fit("Installing Testkube · "+took(elapsed)+" (first run 3 to 8 minutes)", width-5)}
	}
	var rows []string
	for _, s := range p.Services {
		icon, text := liveState(s, spin)
		rows = append(rows, "  "+icon+" "+fmt.Sprintf("%-*s", roleWidth, s.Role)+text(fit(liveText(s), width-5-roleWidth)))
	}
	footer := "  " + spin + " " + fit(downloadedText(p.Downloaded)+fmt.Sprintf("%d/%d ready · ", p.Ready, p.Total)+took(elapsed), width-5)
	// Taller than the window, each redraw would flood scrollback.
	if height > 0 && len(rows)+2 > height {
		return []string{footer}
	}
	return append(rows, footer)
}

func liveState(s localinstall.ServiceProgress, spin string) (icon string, color func(...any) string) {
	plain := func(a ...any) string { return fmt.Sprint(a...) }
	switch s.State {
	case localinstall.StateReady:
		return ui.Green("✔"), plain
	case localinstall.StateDownloading, localinstall.StateStarting:
		return spin, plain
	}
	return ui.LightGray("○"), ui.LightGray
}

func liveText(s localinstall.ServiceProgress) string {
	switch s.State {
	case localinstall.StateReady:
		if s.ReadyAfter > 0 {
			return "ready in " + took(s.ReadyAfter)
		}
		return "ready"
	case localinstall.StateDownloading:
		return "downloading"
	case localinstall.StateQueued:
		return "waiting to download"
	case localinstall.StateWaiting:
		return "waiting for " + strings.ToLower(s.WaitsFor)
	case localinstall.StateNotStarted:
		return "not started yet"
	}
	return "starting"
}

func fit(s string, n int) string {
	r := []rune(s)
	if n < 1 {
		return ""
	}
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}
