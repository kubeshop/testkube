package scheduling

import (
	"sync"
	"time"
)

type executionBatchCursor struct {
	pendingAt   time.Time
	executionID string
}

type executionBatchPager[T any] struct {
	mu             sync.Mutex
	snapshotBefore time.Time
	cursor         *executionBatchCursor
	now            func() time.Time
}

func (p *executionBatchPager[T]) Next(
	fetch func(snapshotBefore time.Time, after *executionBatchCursor) ([]T, error),
	cursorOf func(T) *executionBatchCursor,
) ([]T, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cursor == nil {
		p.snapshotBefore = p.currentTime()
	}

	hadCursor := p.cursor != nil
	items, err := p.fetchPage(fetch, cursorOf)
	if err != nil {
		return nil, err
	}
	if len(items) > 0 {
		return items, nil
	}
	if !hadCursor {
		return nil, nil
	}

	p.snapshotBefore = p.currentTime()
	p.cursor = nil

	return p.fetchPage(fetch, cursorOf)
}

func (p *executionBatchPager[T]) fetchPage(
	fetch func(snapshotBefore time.Time, after *executionBatchCursor) ([]T, error),
	cursorOf func(T) *executionBatchCursor,
) ([]T, error) {
	items, err := fetch(p.snapshotBefore, p.cursor)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}

	p.cursor = cursorOf(items[len(items)-1])
	return items, nil
}

func (p *executionBatchPager[T]) currentTime() time.Time {
	if p.now != nil {
		return p.now().UTC()
	}
	return time.Now().UTC()
}
