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
	mu            sync.Mutex
	advanceCursor *executionBatchCursor
	retryCursor   *executionBatchCursor
	nextRetry     bool
}

func (p *executionBatchPager[T]) Next(
	fetch func(after *executionBatchCursor) ([]T, error),
	cursorOf func(T) *executionBatchCursor,
) ([]T, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.nextRetry {
		items, err := p.fetchPage(&p.retryCursor, fetch, cursorOf)
		if err != nil {
			return nil, err
		}
		if len(items) > 0 {
			p.nextRetry = false
			return items, nil
		}
	}

	items, err := p.fetchPage(&p.advanceCursor, fetch, cursorOf)
	if err != nil {
		return nil, err
	}
	if len(items) > 0 {
		p.nextRetry = true
		return items, nil
	}

	items, err = p.fetchPage(&p.retryCursor, fetch, cursorOf)
	if err != nil {
		return nil, err
	}
	if len(items) > 0 {
		p.nextRetry = false
	}
	return items, nil
}

func (p *executionBatchPager[T]) fetchPage(
	cursor **executionBatchCursor,
	fetch func(after *executionBatchCursor) ([]T, error),
	cursorOf func(T) *executionBatchCursor,
) ([]T, error) {
	items, err := fetch(*cursor)
	if err != nil {
		return nil, err
	}

	if len(items) == 0 && *cursor != nil {
		*cursor = nil
		items, err = fetch(nil)
		if err != nil {
			return nil, err
		}
	}

	if len(items) == 0 {
		return nil, nil
	}

	*cursor = cursorOf(items[len(items)-1])
	return items, nil
}
