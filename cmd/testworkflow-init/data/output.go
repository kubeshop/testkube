package data

import (
	"fmt"
	"os"
)

func Failf(exitCode uint8, message string, args ...interface{}) {
	// Print message
	fmt.Printf(message+"\n", args...)

	// Exit
	os.Exit(int(exitCode))
}
