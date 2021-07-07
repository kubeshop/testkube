package runner

import (
	"io"
)

type Runner interface {
	Run(io.Reader) (Result, error)
}
