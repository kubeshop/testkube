package common

import (
	"os"

	"github.com/mattn/go-isatty"
)

// Gate the org/environment selectors on a TTY. The selector reads stdin, so on a
// non-interactive stream pterm returns immediately and ui.Select discards the
// error, yielding an empty id that only fails much later with an opaque message.
// Var (not func) so tests can override.
var selectorInteractive = func() bool {
	fd := os.Stdin.Fd()
	return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
}
