package scheduling

import (
	"sync"
	"time"
)

type executionBatchCursor struct {
	scheduledAt time.Time
	executionID string
}

type executionBatchPager[T any] struct {
	mu     sync.Mutex
	cursor *executionBatchCursor
}

func (p *executionBatchPager[T]) Next(
	fetch func(after *executionBatchCursor) ([]T, error),
	cursorOf func(T) *executionBatchCursor,
) ([]T, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	items, err := fetch(p.cursor)
	if err != nil {
		return nil, err
	}

	if len(items) == 0 && p.cursor != nil {
		p.cursor = nil
		items, err = fetch(nil)
		if err != nil {
			return nil, err
		}
	}

	if len(items) == 0 {
		return nil, nil
	}

	p.cursor = cursorOf(items[len(items)-1])
	return items, nil
}
