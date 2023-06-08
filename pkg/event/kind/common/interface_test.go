package common

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func TestCompareListeners(t *testing.T) {
	t.Parallel()

	t.Run("both nil metada", func(t *testing.T) {
		t.Parallel()
		l1 := &NilListener{}
		l2 := &NilListener{}

		result := CompareListeners(l1, l2)

		assert.Equal(t, true, result)
	})

	t.Run("one nil metada and one not nil metada", func(t *testing.T) {
		t.Parallel()
		l1 := &NilListener{}
		l2 := &FakeListener{}

		result := CompareListeners(l1, l2)

		assert.Equal(t, false, result)
	})

	t.Run("equal metada", func(t *testing.T) {
		t.Parallel()
		l1 := &FakeListener{field1: "1", field2: "2"}
		l2 := &FakeListener{field1: "1", field2: "2"}

		result := CompareListeners(l1, l2)

		assert.Equal(t, true, result)
	})

	t.Run("not equal metada", func(t *testing.T) {
		t.Parallel()
		l1 := &FakeListener{field1: "2", field2: "1"}
		l2 := &FakeListener{field1: "1", field2: "2"}

		result := CompareListeners(l1, l2)

		assert.Equal(t, false, result)
	})

}

var _ Listener = (*NilListener)(nil)

type NilListener struct {
}

func (l *NilListener) Notify(event testkube.Event) testkube.EventResult {
	return testkube.EventResult{Id: event.Id}
}

func (l *NilListener) Name() string {
	return ""
}

func (l *NilListener) Events() []testkube.EventType {
	return nil
}

func (l NilListener) Selector() string {
	return ""
}

func (l *NilListener) Kind() string {
	return ""
}

func (l *NilListener) Metadata() map[string]string {
	return nil
}

var _ Listener = (*FakeListener)(nil)

type FakeListener struct {
	field1 string
	field2 string
}

func (l *FakeListener) Notify(event testkube.Event) testkube.EventResult {
	return testkube.EventResult{Id: event.Id}
}

func (l *FakeListener) Name() string {
	return ""
}

func (l *FakeListener) Events() []testkube.EventType {
	return nil
}

func (l FakeListener) Selector() string {
	return ""
}

func (l *FakeListener) Kind() string {
	return ""
}

func (l *FakeListener) Metadata() map[string]string {
	return map[string]string{
		"field1": l.field1,
		"field2": l.field2,
	}
}
