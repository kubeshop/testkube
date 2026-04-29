package common

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUIShell_SuppressedWhenStdoutNotTTY(t *testing.T) {
	cases := []struct {
		name string
		fn   func()
	}{
		{"GetExecution", func() { UIShellGetExecution("test-id") }},
		{"ViewExecution", func() { UIShellViewExecution("test-id") }},
		{"WatchExecution", func() { UIShellWatchExecution("test-id") }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := captureStdout(t, tc.fn)
			require.Empty(t, out, "hint must not be written when stdout is not a TTY")
		})
	}
}

// captureStdout swaps os.Stdout for an os.Pipe (non-tty fd) so hintsEnabled() returns false.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	require.NoError(t, err)

	saved := os.Stdout
	os.Stdout = w
	t.Cleanup(func() {
		os.Stdout = saved
		_ = r.Close()
	})

	done := make(chan string, 1)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()

	fn()
	require.NoError(t, w.Close())
	return <-done
}
