// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

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
