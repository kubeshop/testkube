// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

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
