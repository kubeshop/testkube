package rand

import (
	petname "github.com/dustinkirkland/golang-petname"
)

func Name() string {
	return petname.Generate(3, "-")
}
