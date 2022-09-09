package dummy

import "github.com/kubeshop/testkube/pkg/event/kind/common"

type DummyLoader struct {
	Err error
}

func (r DummyLoader) Kind() string {
	return "dummy"
}

func (r *DummyLoader) Load() (common.Listeners, error) {
	if r.Err != nil {
		return nil, r.Err
	}
	return common.Listeners{
		&DummyListener{},
		&DummyListener{},
	}, nil
}
