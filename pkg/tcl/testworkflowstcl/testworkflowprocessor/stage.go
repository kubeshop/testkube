// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package testworkflowprocessor

import (
	"github.com/kubeshop/testkube/pkg/imageinspector"
	"github.com/kubeshop/testkube/pkg/tcl/expressionstcl"
)

//go:generate mockgen -destination=./mock_stage.go -package=testworkflowprocessor "github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/testworkflowprocessor" Stage
type Stage interface {
	StageMetadata
	StageLifecycle
	Len() int
	Signature() Signature
	Resolve(m ...expressionstcl.Machine) error
	ContainerStages() []ContainerStage
	GetImages() map[string]struct{}
	ApplyImages(images map[string]*imageinspector.Info) error
	Flatten() []Stage
}
