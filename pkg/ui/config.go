package ui

import (
	"io"
	"os"
)

// Verbose adds additional info messages e.g. in case of checking errors
var Verbose = false

var Writer io.Writer = os.Stdout
