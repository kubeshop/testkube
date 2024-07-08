package data

import (
	"fmt"
	"os"
)

func Failf(exitCode uint8, message string, args ...interface{}) {
	// Print message
	fmt.Printf(message+"\n", args...)

	// Kill the sub-process
	Step.Kill()

	// Exit
	os.Exit(int(exitCode))
}
