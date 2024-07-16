package stage

import (
	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/imageinspector"
)

type containerStage struct {
	stageMetadata
	stageLifecycle
	container Container
}

type ContainerStage interface {
	Stage
	Container() Container
}

func NewContainerStage(ref string, container Container) ContainerStage {
	return &containerStage{
		stageMetadata: stageMetadata{ref: ref},
		container:     container.CreateChild(),
	}
}

func (s *containerStage) Len() int {
	return 1
}

func (s *containerStage) Signature() Signature {
	return &signature{
		RefValue:      s.ref,
		NameValue:     s.name,
		CategoryValue: s.category,
		OptionalValue: s.optional,
		NegativeValue: s.negative,
		ChildrenValue: nil,
	}
}

func (s *containerStage) ContainerStages() []ContainerStage {
	return []ContainerStage{s}
}

func (s *containerStage) GetImages() map[string]struct{} {
	return map[string]struct{}{s.container.Image(): {}}
}

func (s *containerStage) Flatten() []Stage {
	return []Stage{s}
}

func (s *containerStage) ApplyImages(images map[string]*imageinspector.Info, imageNameResolutions map[string]string) error {
	originalImageName := s.container.Image()
	return s.container.ApplyImageData(images[originalImageName], imageNameResolutions[originalImageName])
}

func (s *containerStage) Resolve(m ...expressions.Machine) error {
	err := s.container.Resolve(m...)
	if err != nil {
		return errors.Wrap(err, "stage container")
	}
	return expressions.Simplify(s, m...)
}

func (s *containerStage) Container() Container {
	return s.container
}

func (s *containerStage) HasPause() bool {
	return s.paused
}
