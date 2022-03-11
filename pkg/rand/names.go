package rand

import (
	petname "github.com/dustinkirkland/golang-petname"
)

// Return random name similar to Docker separated by `-`
func Name() string {
	return petname.Generate(3, "-")
}
