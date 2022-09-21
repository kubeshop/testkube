package dummy

import (
	"fmt"

	"github.com/kubeshop/testkube/pkg/event/kind/common"
)

type DummyLoader struct {
	IdPrefix string
	Err      error
}

func (r DummyLoader) Kind() string {
	return "dummy"
}

func (r *DummyLoader) Load() (common.Listeners, error) {
	if r.Err != nil {
		return nil, r.Err
	}
	return common.Listeners{
		&DummyListener{Id: r.name(1)},
		&DummyListener{Id: r.name(2)},
	}, nil
}

func (r *DummyLoader) name(i int) string {
	return fmt.Sprintf("%s.%d", r.IdPrefix, i)
}
