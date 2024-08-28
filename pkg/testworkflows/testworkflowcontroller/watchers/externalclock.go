package watchers

//import (
//	"sync"
//	"time"
//)
//
//type externalClockPair struct {
//	external time.Time
//	local    time.Time
//}
//
//func (e *externalClockPair) MinExternalAt(localTs time.Time) time.Time {
//	return e.external.Add(localTs.Sub(e.local))
//}
//
//type externalClock struct {
//	pairs  []externalClockPair
//	oldest int
//	mu     sync.RWMutex
//}
//
//// ExternalClock helps to compare current time (full precision) with external system (seconds precision), like Kubernetes
//type ExternalClock interface {
//	Ready() bool
//	RegisterAt(externalTsInPast, localTs time.Time)
//	After(externalTsInPast, localTs time.Time) bool
//	Before(externalTsInPast, localTs time.Time) bool
//}
//
//func (e *externalClock) Ready() bool {
//	e.mu.RLock()
//	defer e.mu.RUnlock()
//	return len(e.pairs) > 0
//}
//
//func (e *externalClock) RegisterAt(externalTs, localTs time.Time) {
//	e.mu.Lock()
//	defer e.mu.Unlock()
//
//	// Check if there is such external timestamp already in the list,
//	// and replace with the time that is more in the past,
//	// for more accurate values.
//	for i := range e.pairs {
//		if e.pairs[i].external.Equal(externalTs) {
//			if e.pairs[i].local.After(localTs) {
//				e.pairs[i].local = localTs
//			}
//			return
//		}
//	}
//
//	// Otherwise, append it to the list, or replace the oldest entry
//	if len(e.pairs) < cap(e.pairs) {
//		e.pairs = append(e.pairs, externalClockPair{local: localTs, external: externalTs})
//	} else {
//		e.pairs[e.oldest] = externalClockPair{local: localTs, external: externalTs}
//		e.oldest = (e.oldest + 1) % len(e.pairs)
//	}
//}
//
//func (e *externalClock) After(externalTsInPast, localTs time.Time) bool {
//	e.mu.RLock()
//	defer e.mu.RUnlock()
//
//	// Ensure that we fill the whole second space, as we don't know what that external TS is.
//	// 2012-09-09T12:34:00.000000000Z becomes 2012-09-09T12:34:00.999999999Z).
//	externalTsInPast = externalTsInPast.Truncate(time.Second).Add(time.Second - 1)
//
//	for i := range e.pairs {
//		if e.pairs[i].MinExternalAt(localTs).After(externalTsInPast) {
//			return true
//		}
//	}
//	return false
//}
//
//func (e *externalClock) Before(externalTsInPast, localTs time.Time) bool {
//	e.mu.RLock()
//	defer e.mu.RUnlock()
//
//	// Ensure that we fill the whole second space, as we don't know what that external TS is.
//	// 2012-09-09T12:34:00.000000000Z becomes 2012-09-09T12:34:00.999999999Z).
//	externalTsInPast = externalTsInPast.Truncate(time.Second)
//
//	for i := range e.pairs {
//		if e.pairs[i].MinExternalAt(localTs).Before(externalTsInPast) {
//			return true
//		}
//	}
//	return false
//}
//
//func NewExternalClock(buffer int) ExternalClock {
//	return &externalClock{pairs: make([]externalClockPair, 0, buffer)}
//}
