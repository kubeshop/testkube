package dummy

import (
	"fmt"

	"github.com/kubeshop/testkube/pkg/event/kind/common"
)

type DummyLoader struct {
	IdPrefix          string
	Err               error
	SelectorString    string
	ListenersOverride []common.Listener
}

func (r DummyLoader) Kind() string {
	return "dummy"
}

func (r *DummyLoader) Load() (common.Listeners, error) {
	if r.Err != nil {
		return nil, r.Err
	}
	if r.ListenersOverride != nil {
		return r.ListenersOverride, nil
	}
	return common.Listeners{
		&DummyListener{Id: r.name(1), SelectorString: r.SelectorString},
		&DummyListener{Id: r.name(2), SelectorString: r.SelectorString},
	}, nil
}

func (r *DummyLoader) name(i int) string {
	return fmt.Sprintf("%s.%d", r.IdPrefix, i)
}
