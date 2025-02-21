package testworkflowprocessor

import (
	"fmt"
	"strconv"
	"sync/atomic"

	"k8s.io/apimachinery/pkg/util/rand"
)

type RefCounter interface {
	NextRef() string
}

type refCounter struct {
	refCount atomic.Uint64
}

func NewRefCounter() RefCounter {
	return &refCounter{}
}

func (r *refCounter) NextRef() string {
	next := r.refCount.Add(1)
	return fmt.Sprintf("r%s%s", rand.String(5), strconv.FormatUint(next, 36))
}
