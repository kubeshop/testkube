package rand

import (
	"math/rand"
	"time"

	petname "github.com/dustinkirkland/golang-petname"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func Name() string {
	return petname.Generate(3, "-")
}
