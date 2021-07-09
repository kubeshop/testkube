package runner

import (
	"io"
)

// Runner interface to abstract runners implementations
type Runner interface {
	// Run returns output as string (for now probably we could have other needs?)
	Run(io.Reader) (string, error)
}
