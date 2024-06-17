package artifacts

import (
	"io"
)

type Uploader interface {
	Start() error
	Add(path string, file io.Reader, size int64) error
	End() error
}
