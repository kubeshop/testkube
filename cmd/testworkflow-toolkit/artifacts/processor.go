package artifacts

import "io/fs"

type Processor interface {
	Start() error
	Add(uploader Uploader, path string, file fs.File, stat fs.FileInfo) error
	End() error
}

type PostProcessor interface {
	Start() error
	Add(path string) error
	End() error
}
