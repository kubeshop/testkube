package stage

import (
	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/imageinspector"
)

//go:generate go tool mockgen -destination=./mock_stage.go -package=stage "github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/stage" Stage
type Stage interface {
	StageMetadata
	StageLifecycle
	Len() int
	HasPause() bool
	Signature() Signature
	FullSignature() Signature
	Resolve(m ...expressions.Machine) error
	ContainerStages() []ContainerStage
	GetImages(isGroupNeeded bool) map[string]bool
	ApplyImages(images map[string]*imageinspector.Info, imageNameResolutions map[string]string) error
	Flatten() []Stage
}
