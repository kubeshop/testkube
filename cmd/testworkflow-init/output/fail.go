package output

import "os"

func ExitErrorf(exitCode uint8, message string, args ...interface{}) {
	// Print message
	Std.Printf(message+"\n", args...)

	// Exit
	os.Exit(int(exitCode))
}

func UnsafeExitErrorf(exitCode uint8, message string, args ...interface{}) {
	// Print message
	Std.Direct().Printf(message+"\n", args...)

	// Exit
	os.Exit(int(exitCode))
}
