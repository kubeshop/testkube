package rand

import (
	petname "github.com/dustinkirkland/golang-petname"
)

// Name return random 3 part string similar to Docker image names, separated by `-`
func Name() string {
	return petname.Generate(3, "-")
}
