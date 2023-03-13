package slack

import "github.com/kubeshop/testkube/pkg/api/v1/testkube"

var exists = struct{}{}

type set[S Storable] struct {
	m map[S]struct{}
}

type Storable interface {
	string | testkube.EventType
}

func NewSet[S Storable]() *set[S] {
	s := &set[S]{}
	s.m = make(map[S]struct{})
	return s
}

func NewSetFromArray[S Storable](s []S) *set[S] {
	set := NewSet[S]()
	for _, v := range s {
		set.Add(v)
	}
	return set
}

func (s *set[S]) Add(value S) {
	s.m[value] = exists
}

func (s *set[S]) Remove(value S) {
	delete(s.m, value)
}

func (s *set[S]) Contains(value S) bool {
	_, c := s.m[value]
	return c
}

func (s *set[S]) IsEmpty() bool {
	return len(s.m) == 0
}
