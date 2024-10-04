package stage

type StageMetadata interface {
	Ref() string
	Name() string
	Category() string

	SetName(name string) StageMetadata
	SetCategory(category string) StageMetadata
}

type stageMetadata struct {
	ref      string
	name     string
	category string
}

func NewStageMetadata(ref string) StageMetadata {
	return &stageMetadata{ref: ref}
}

func (s *stageMetadata) Ref() string {
	return s.ref
}

func (s *stageMetadata) Name() string {
	return s.name
}

func (s *stageMetadata) Category() string {
	return s.category
}

func (s *stageMetadata) SetName(name string) StageMetadata {
	s.name = name
	return s
}

func (s *stageMetadata) SetCategory(category string) StageMetadata {
	s.category = category
	return s
}
