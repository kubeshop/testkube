// Package ui prints CLI output and asks the user for input.
//
// # Prompts without a terminal
//
// interactive.go holds the terminal check every prompt goes through. Select, Confirm and
// TextInput call requireInteractive first and exit with an actionable message when stdin is not
// a terminal, so no call site has to guard itself. StdinIsInteractive exposes the same check for
// callers that want to take a different path instead of failing.
//
// The guard exists because a prompt with nobody to answer it used to spin: atomicgo.dev/keyboard
// cannot open a non-terminal stdin, reports every failed read as an empty keypress with a nil
// error, and pterm has no case for an empty keypress, so its listener loops forever and burns
// more than a core.
//
// A new prompt uses these helpers rather than pterm directly, and inherits the guard. A command
// that must not prompt at all branches on StdinIsInteractive. Tests flip the unexported
// stdinIsInteractive seam rather than allocating a pseudo terminal.
package ui
