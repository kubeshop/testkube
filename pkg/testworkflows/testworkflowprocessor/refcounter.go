package testworkflowprocessor

import (
	"fmt"
	"strconv"

	"k8s.io/apimachinery/pkg/util/rand"
)

type RefCounter interface {
	NextRef() string
}

type refCounter struct {
	refCount uint64
}

func NewRefCounter() RefCounter {
	return &refCounter{}
}

func (r *refCounter) NextRef() string {
	return fmt.Sprintf("r%s%s", rand.String(5), strconv.FormatUint(r.refCount, 36))
}
