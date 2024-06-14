// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package testworkflowprocessor

import (
	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/imageinspector"
)

//go:generate mockgen -destination=./mock_stage.go -package=testworkflowprocessor "github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor" Stage
type Stage interface {
	StageMetadata
	StageLifecycle
	Len() int
	HasPause() bool
	Signature() Signature
	Resolve(m ...expressions.Machine) error
	ContainerStages() []ContainerStage
	GetImages() map[string]struct{}
	ApplyImages(images map[string]*imageinspector.Info) error
	Flatten() []Stage
}
