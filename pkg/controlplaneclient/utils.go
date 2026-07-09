package controlplaneclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
)

var (
	grpcOpts = []grpc.CallOption{grpc.UseCompressor(gzip.Name), grpc.MaxCallRecvMsgSize(math.MaxInt32)}
)

// TODO: add timeout?
func call[Request any, Response any](ctx context.Context, md metadata.MD, fn func(context.Context, Request, ...grpc.CallOption) (Response, error), req Request) (Response, error) {
	if ctx.Err() != nil {
		var v Response
		return v, ctx.Err()
	}
	return fn(metadata.NewOutgoingContext(ctx, md), req, grpcOpts...)
}

// TODO: add timeout?
func watch[Response any](ctx context.Context, md metadata.MD, fn func(context.Context, ...grpc.CallOption) (Response, error)) (Response, error) {
	if ctx.Err() != nil {
		var v Response
		return v, ctx.Err()
	}
	return fn(metadata.NewOutgoingContext(ctx, md), grpcOpts...)
}

func getGrpcErrorCode(err error) codes.Code {
	if err == nil {
		return codes.Unknown
	}
	if e, ok := err.(interface{ GRPCStatus() *status.Status }); ok {
		return e.GRPCStatus().Code()
	}
	return codes.Unknown
}

type notificationRequest interface {
	GetRequestType() cloud.TestWorkflowNotificationsRequestType
	GetStreamId() string
	GetResumeAfterSeqNo() uint32
}

type notificationSrv[Request any, Response any] interface {
	Send(Response) error
	Recv() (Request, error)
}

const (
	workflowNotificationHeartbeatInterval = 20 * time.Second
	workflowNotificationReplayMaxEvents   = 10_000
	workflowNotificationReplayMaxBytes    = 10 * 1024 * 1024
	workflowNotificationSessionIdleTTL    = 15 * time.Minute
	// defaultLiveLogReplayMaxBytes is the aggregate replay budget shared by the
	// workflow, parallel and service managers when the operator does not set one.
	defaultLiveLogReplayMaxBytes = 256 * 1024 * 1024
)

// liveLogReplayBudget bounds the total bytes held across every replay buffer of
// every notification stream session, on top of the per-session inner caps. When
// used would exceed max, registered evictors reclaim bytes: first by dropping
// done sessions, then by trimming the oldest replay events of running sessions.
//
// Lock ordering: manager.mu -> session.mu -> budget.mu. budget.mu is a leaf:
// reserve/release take only budget.mu and return immediately, never invoking an
// evictor or acquiring manager/session locks. A session reserves its bytes while
// still holding session.mu, so a growth is counted atomically with the buffer
// mutation and can never be released before it is reserved. The eviction trigger
// runs only after session.mu is dropped; evictors acquire manager.mu then
// session.mu. This keeps publish non-blocking and deadlock-free.
type liveLogReplayBudget struct {
	mu            sync.Mutex
	used          int64
	max           int64
	nextEvictorID uint64
	evictors      map[uint64]func(need int64, exclude *notificationStreamSession) int64
}

func newLiveLogReplayBudget(max int64) *liveLogReplayBudget {
	if max <= 0 {
		max = defaultLiveLogReplayMaxBytes
	}
	return &liveLogReplayBudget{
		max:      max,
		evictors: make(map[uint64]func(need int64, exclude *notificationStreamSession) int64),
	}
}

// register adds a manager's evictor and returns a deregister func that removes it.
// The budget is shared and long-lived while managers come and go with each gRPC
// connection, so the caller must call deregister when its manager tears down or
// the evictor set grows without bound. deregister takes only budget.mu (leaf).
func (b *liveLogReplayBudget) register(evict func(need int64, exclude *notificationStreamSession) int64) func() {
	if b == nil {
		return func() {}
	}
	b.mu.Lock()
	id := b.nextEvictorID
	b.nextEvictorID++
	b.evictors[id] = evict
	b.mu.Unlock()
	return func() {
		b.mu.Lock()
		delete(b.evictors, id)
		b.mu.Unlock()
	}
}

// reserve accounts a net change to the caller's replay buffer against used. It is
// a pure leaf: it takes only budget.mu and returns immediately, never invoking an
// evictor or acquiring manager/session locks. Callers reserve while still holding
// their session.mu, so the reservation is atomic with the buffer mutation and a
// growth can never be released before it is reserved. delta may be negative when a
// mutation nets to a shrink (e.g. inner-cap eviction outweighs the append).
func (b *liveLogReplayBudget) reserve(delta int64) {
	if b == nil || delta == 0 {
		return
	}
	b.mu.Lock()
	b.used += delta
	if b.used < 0 {
		b.used = 0
	}
	b.mu.Unlock()
}

// evictIfOver reclaims bytes when used exceeds max. It must be called only after
// the caller has released its session.mu, since evictors acquire manager.mu then
// session.mu. caller is passed as exclude so an evictor trims the caller last, via
// the caller's own (now-released) lock. evictIfOver never blocks on a held lock.
func (b *liveLogReplayBudget) evictIfOver(caller *notificationStreamSession) {
	if b == nil {
		return
	}
	b.mu.Lock()
	over := b.used - b.max
	if over <= 0 {
		b.mu.Unlock()
		return
	}
	evictors := make([]func(int64, *notificationStreamSession) int64, 0, len(b.evictors))
	for _, evict := range b.evictors {
		evictors = append(evictors, evict)
	}
	b.mu.Unlock()

	for _, evict := range evictors {
		if over <= 0 {
			break
		}
		freed := evict(over, caller)
		// Freed bytes are released against the budget by the eviction path
		// itself (via release), so recompute remaining overage from the budget.
		if freed > 0 {
			b.mu.Lock()
			over = b.used - b.max
			b.mu.Unlock()
		}
	}
}

// release returns bytes to the budget when a replay buffer shrinks (inner-cap
// eviction, trim, or session drop). Releases must exactly match prior grows.
func (b *liveLogReplayBudget) release(delta int64) {
	if b == nil || delta <= 0 {
		return
	}
	b.mu.Lock()
	b.used -= delta
	if b.used < 0 {
		b.used = 0
	}
	b.mu.Unlock()
}

type notificationStreamEvent struct {
	seqNo        uint32
	notification *testkube.TestWorkflowExecutionNotification
}

func notificationResumable(notification *testkube.TestWorkflowExecutionNotification) bool {
	if notification == nil || notification.Temporary {
		return false
	}
	return notification.Result != nil || notification.Output != nil || notification.Log != ""
}

type notificationStreamSubscription struct {
	id        uint64
	ch        chan notificationStreamEvent
	done      chan struct{}
	closeOnce sync.Once
}

func newNotificationStreamSubscription(id uint64) *notificationStreamSubscription {
	return &notificationStreamSubscription{
		id:   id,
		ch:   make(chan notificationStreamEvent, 256),
		done: make(chan struct{}),
	}
}

func (s *notificationStreamSubscription) send(event notificationStreamEvent) {
	select {
	case <-s.done:
		return
	default:
	}
	select {
	case s.ch <- event:
	case <-s.done:
	}
}

func (s *notificationStreamSubscription) close() {
	s.closeOnce.Do(func() {
		close(s.done)
	})
}

type notificationStreamSession struct {
	mu          sync.Mutex
	budget      *liveLogReplayBudget
	nextSeqNo   uint32
	replay      []notificationStreamEvent
	replayBytes int
	subscribers map[uint64]*notificationStreamSubscription
	done        bool
	errored     bool
	lastSeqNo   uint32
	lastActive  time.Time
}

func newNotificationStreamSession(budget *liveLogReplayBudget) *notificationStreamSession {
	return &notificationStreamSession{
		budget:      budget,
		nextSeqNo:   1,
		subscribers: make(map[uint64]*notificationStreamSubscription),
		lastActive:  time.Now(),
	}
}

func (s *notificationStreamSession) subscribe(resumeAfterSeqNo uint32, subscriptionID uint64) (*notificationStreamSubscription, []notificationStreamEvent, bool, uint32, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.lastActive = time.Now()
	sub := newNotificationStreamSubscription(subscriptionID)
	if !s.done {
		s.subscribers[sub.id] = sub
	}

	replay, available := s.replayAfterLocked(resumeAfterSeqNo)
	return sub, replay, available, s.lastSeqNo, s.done
}

func (s *notificationStreamSession) unsubscribe(sub *notificationStreamSubscription) {
	s.mu.Lock()
	removed, ok := s.subscribers[sub.id]
	if ok {
		delete(s.subscribers, sub.id)
	}
	s.mu.Unlock()
	if ok {
		removed.close()
	}
}

func (s *notificationStreamSession) publish(notification *testkube.TestWorkflowExecutionNotification) {
	if notification == nil {
		return
	}

	var subscribers []*notificationStreamSubscription
	var grownBytes int
	var freedBytes int
	s.mu.Lock()
	s.lastActive = time.Now()
	seqNo := uint32(0)
	if notificationResumable(notification) {
		seqNo = s.nextSeqNo
		s.nextSeqNo++
		s.lastSeqNo = seqNo
	}

	event := notificationStreamEvent{
		seqNo:        seqNo,
		notification: notification,
	}
	if seqNo > 0 {
		added := approximateNotificationBytes(notification)
		s.replay = append(s.replay, event)
		s.replayBytes += added
		grownBytes = added
		for len(s.replay) > workflowNotificationReplayMaxEvents || s.replayBytes > workflowNotificationReplayMaxBytes {
			evicted := approximateNotificationBytes(s.replay[0].notification)
			s.replayBytes -= evicted
			freedBytes += evicted
			s.replay[0].notification = nil
			s.replay = s.replay[1:]
		}
		// Reserve the net change while still holding s.mu, so the reservation is
		// atomic with the buffer mutation. A concurrent evictor can then only
		// release bytes it has already seen reserved, never a phantom growth.
		s.budget.reserve(int64(grownBytes - freedBytes))
	}

	for _, sub := range s.subscribers {
		subscribers = append(subscribers, sub)
	}
	s.mu.Unlock()

	// Trigger eviction outside s.mu so evictors never re-enter this session's lock.
	s.budget.evictIfOver(s)

	for _, sub := range subscribers {
		sub.send(event)
	}
}

func (s *notificationStreamSession) close(errored bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.done {
		return
	}
	s.done = true
	s.errored = errored
	s.lastActive = time.Now()
	for id, sub := range s.subscribers {
		sub.close()
		delete(s.subscribers, id)
	}
}

// dropReplay discards the whole replay buffer and returns the bytes it held, so
// the caller can release them against the budget. Used when evicting a done session.
func (s *notificationStreamSession) dropReplay() int64 {
	s.mu.Lock()
	freed := s.replayBytes
	for i := range s.replay {
		s.replay[i].notification = nil
	}
	s.replay = nil
	s.replayBytes = 0
	s.mu.Unlock()
	return int64(freed)
}

// trimReplay drops oldest replay events until at least need bytes are freed or
// the buffer is empty, returning the bytes freed. Lowers resume depth for running
// sessions under budget pressure; a later resume past the trimmed point returns
// resume_unavailable downstream.
func (s *notificationStreamSession) trimReplay(need int64) int64 {
	if need <= 0 {
		return 0
	}
	s.mu.Lock()
	var freed int64
	for freed < need && len(s.replay) > 0 {
		evicted := approximateNotificationBytes(s.replay[0].notification)
		s.replayBytes -= evicted
		freed += int64(evicted)
		s.replay[0].notification = nil
		s.replay = s.replay[1:]
	}
	s.mu.Unlock()
	return freed
}

func (s *notificationStreamSession) currentSeqNo() uint32 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastSeqNo
}

func (s *notificationStreamSession) status() (done bool, errored bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.done, s.errored
}

func (s *notificationStreamSession) replayAfterLocked(resumeAfterSeqNo uint32) ([]notificationStreamEvent, bool) {
	if resumeAfterSeqNo == 0 {
		return nil, true
	}
	if len(s.replay) == 0 {
		return nil, resumeAfterSeqNo >= s.lastSeqNo
	}

	earliest := s.replay[0].seqNo
	if resumeAfterSeqNo < earliest-1 {
		return nil, false
	}

	var replay []notificationStreamEvent
	for _, event := range s.replay {
		if event.seqNo > resumeAfterSeqNo {
			replay = append(replay, event)
		}
	}
	return replay, true
}

func (s *notificationStreamSession) expired(now time.Time, ttl time.Duration) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.done && now.Sub(s.lastActive) >= ttl
}

func approximateNotificationBytes(notification *testkube.TestWorkflowExecutionNotification) int {
	if notification == nil {
		return 0
	}
	b, err := json.Marshal(notification)
	if err != nil {
		return len(notification.Log)
	}
	return len(b)
}

func sendNotificationResponse[Response any](ctx context.Context, responses chan<- Response, response Response) error {
	select {
	case responses <- response:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

type notificationStreamSessionManager[Request notificationRequest] struct {
	mu             sync.Mutex
	nextID         atomic.Uint64
	sessions       map[string]*notificationStreamSession
	sessionIdleTTL time.Duration
	budget         *liveLogReplayBudget
	deregister     func()
	key            func(Request) string
	process        func(ctx context.Context, req Request) NotificationWatcher
}

func newNotificationStreamSessionManager[Request notificationRequest](
	budget *liveLogReplayBudget,
	key func(Request) string,
	process func(ctx context.Context, req Request) NotificationWatcher,
) *notificationStreamSessionManager[Request] {
	m := &notificationStreamSessionManager[Request]{
		sessions:       make(map[string]*notificationStreamSession),
		sessionIdleTTL: workflowNotificationSessionIdleTTL,
		budget:         budget,
		key:            key,
		process:        process,
	}
	m.deregister = budget.register(m.evict)
	return m
}

// stop removes this manager's evictor from the shared budget. It must be called
// when the manager's connection ends so evictors do not accumulate across
// reconnects. Orphaned sessions still release their own bytes via idle-TTL expiry.
func (m *notificationStreamSessionManager[Request]) stop() {
	if m.deregister != nil {
		m.deregister()
	}
}

// evict reclaims up to need bytes for the budget: first by dropping done sessions
// from the map, then by trimming the oldest replay events of running sessions.
// It releases every freed byte against the budget itself. Acquires manager.mu then
// each session's lock (via the session helpers); budget.mu is not held on entry.
func (m *notificationStreamSessionManager[Request]) evict(need int64, exclude *notificationStreamSession) int64 {
	var freed int64

	// Pass 1: drop done sessions entirely.
	m.mu.Lock()
	for key, session := range m.sessions {
		if freed >= need {
			break
		}
		if done, _ := session.status(); !done {
			continue
		}
		delete(m.sessions, key)
		freed += session.dropReplay()
	}
	// Collect remaining running sessions for pass 2 before releasing the lock.
	var running []*notificationStreamSession
	if freed < need {
		running = make([]*notificationStreamSession, 0, len(m.sessions))
		for _, session := range m.sessions {
			running = append(running, session)
		}
	}
	m.mu.Unlock()

	// Release dropped-session bytes; they are gone from the map now.
	m.budget.release(freed)

	// Pass 2: trim oldest events of running sessions until enough is reclaimed.
	for _, session := range running {
		if freed >= need {
			break
		}
		if session == exclude {
			// The caller's own growth is accounted separately; trimming it here
			// too is safe (its lock is released) but redundant, so skip it first
			// and only fall back to it below if others cannot cover the need.
			continue
		}
		t := session.trimReplay(need - freed)
		freed += t
		m.budget.release(t)
	}
	if freed < need && exclude != nil {
		t := exclude.trimReplay(need - freed)
		freed += t
		m.budget.release(t)
	}
	return freed
}

func (m *notificationStreamSessionManager[Request]) sessionKey(req Request) string {
	key := m.key(req)
	if req.GetStreamId() == "" {
		return key
	}
	return fmt.Sprintf("%s:%s", key, req.GetStreamId())
}

// dropSessionLocked removes a session from the map and releases its replay bytes
// against the budget. Caller must hold m.mu.
func (m *notificationStreamSessionManager[Request]) dropSessionLocked(key string, session *notificationStreamSession) {
	delete(m.sessions, key)
	m.budget.release(session.dropReplay())
}

func (m *notificationStreamSessionManager[Request]) attach(ctx context.Context, req Request) (*notificationStreamSession, *notificationStreamSubscription, []notificationStreamEvent, bool, uint32, bool) {
	key := m.sessionKey(req)
	now := time.Now()

	m.mu.Lock()
	for sessionKey, session := range m.sessions {
		if session.expired(now, m.sessionIdleTTL) {
			m.dropSessionLocked(sessionKey, session)
		}
	}

	session := m.sessions[key]
	if req.GetResumeAfterSeqNo() == 0 {
		session = nil
	} else if session != nil {
		done, errored := session.status()
		if done && errored {
			m.dropSessionLocked(key, session)
			session = nil
		}
	}
	freshSession := false
	if session == nil {
		session = newNotificationStreamSession(m.budget)
		m.sessions[key] = session
		freshSession = true
	}
	subscriptionID := m.nextID.Add(1)
	m.mu.Unlock()

	subscribeAfterSeqNo := req.GetResumeAfterSeqNo()
	if freshSession && req.GetResumeAfterSeqNo() > 0 {
		subscribeAfterSeqNo = 0
	}
	sub, replay, available, lastSeqNo, done := session.subscribe(subscribeAfterSeqNo, subscriptionID)
	if freshSession {
		liveOnlyAfter := time.Time{}
		if req.GetResumeAfterSeqNo() > 0 {
			liveOnlyAfter = time.Now()
		}
		go m.runSource(ctx, key, session, req, liveOnlyAfter)
		if req.GetResumeAfterSeqNo() > 0 {
			available = false
		}
	}
	return session, sub, replay, available, lastSeqNo, done
}

func (m *notificationStreamSessionManager[Request]) scheduleExpiration(key string, session *notificationStreamSession) {
	time.AfterFunc(m.sessionIdleTTL, func() {
		m.mu.Lock()
		defer m.mu.Unlock()

		if m.sessions[key] != session {
			return
		}
		if session.expired(time.Now(), m.sessionIdleTTL) {
			m.dropSessionLocked(key, session)
		}
	})
}

func shouldPublishLiveResumeNotification(notification *testkube.TestWorkflowExecutionNotification, attachedAt time.Time) bool {
	if !notificationResumable(notification) {
		return false
	}
	if notification.Ts.IsZero() {
		return false
	}
	return !notification.Ts.Before(attachedAt)
}

func (m *notificationStreamSessionManager[Request]) runSource(ctx context.Context, key string, session *notificationStreamSession, req Request, liveOnlyAfter time.Time) {
	var sourceErr error
	defer func() {
		session.close(sourceErr != nil)
		m.scheduleExpiration(key, session)
	}()

	watcher := m.process(ctx, req)
	for notification := range watcher.Channel() {
		if !liveOnlyAfter.IsZero() && !shouldPublishLiveResumeNotification(notification, liveOnlyAfter) {
			continue
		}
		session.publish(notification)
	}
	sourceErr = watcher.Err()
	if sourceErr != nil {
		session.publish(&testkube.TestWorkflowExecutionNotification{
			Ts:        time.Now(),
			EventType: "error",
			Log:       fmt.Sprintf("%s %s", time.Now().Format(constants.PreciseTimeFormat), sourceErr.Error()),
		})
	}
}

func processNotifications[Request notificationRequest, Response any, Srv notificationSrv[Request, Response]](
	ctx context.Context,
	md metadata.MD,
	fn func(context.Context, ...grpc.CallOption) (Srv, error),
	buildPongNotification func(streamId string) Response,
	buildNotification func(streamId string, seqNo uint32, notification *testkube.TestWorkflowExecutionNotification) Response,
	buildError func(streamId string, message string) Response,
	buildProtocol func(streamId string, seqNo uint32, notificationType cloud.TestWorkflowNotificationType, message string) Response,
	sessionKey func(req Request) string,
	process func(ctx context.Context, req Request) NotificationWatcher,
	budget *liveLogReplayBudget,
	sendTimeout time.Duration,
	recvTimeout time.Duration,
	logger *zap.SugaredLogger,
) error {
	g, ctx := errgroup.WithContext(ctx)
	stream, err := watch(ctx, md, fn)
	if err != nil {
		return err
	}

	responses := make(chan Response, 5)
	sendResponse := func(response Response) error {
		return sendNotificationResponse(ctx, responses, response)
	}
	sessionManager := newNotificationStreamSessionManager(budget, sessionKey, process)
	// Remove this connection's evictor from the shared budget when the stream ends,
	// so evictor registrations do not accumulate across reconnects.
	defer sessionManager.stop()

	// Send responses in sequence
	// GRPC stream have special requirements for concurrency on SendMsg, and RecvMsg calls.
	// Please check https://github.com/grpc/grpc-go/blob/master/Documentation/concurrency.md
	g.Go(func() error {
		for msg := range responses {
			errChan := make(chan error, 1)
			go func() {
				errChan <- stream.Send(msg)
				close(errChan)
			}()

			t := time.NewTimer(sendTimeout)

			select {
			case err := <-errChan:
				t.Stop()
				if err != nil {
					return err
				}
			case <-ctx.Done():
				t.Stop()
				return ctx.Err()
			case <-t.C:
				return fmt.Errorf("send response too slow")
			}
		}
		return nil
	})

	// Process the requests
	g.Go(func() error {
		var wg sync.WaitGroup
		defer func() {
			wg.Wait()
			close(responses)
		}()
		for {
			// Take the context error if possible
			if err == nil && ctx.Err() != nil {
				err = ctx.Err()
			}

			// Handle the error
			if err != nil {
				return err
			}

			// Get the next request
			var req Request
			reqChan := make(chan struct {
				req Request
				err error
			}, 1)
			go func() {
				recvReq, recvErr := stream.Recv()
				reqChan <- struct {
					req Request
					err error
				}{recvReq, recvErr}
			}()

			select {
			case result := <-reqChan:
				req = result.req
				err = result.err
				if err != nil {
					return err
				}
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(recvTimeout):
				err = errors.New("receive request too slow")
				return err
			}

			// Send PONG to the PING message
			if req.GetRequestType() == cloud.TestWorkflowNotificationsRequestType_WORKFLOW_STREAM_HEALTH_CHECK {
				if err = sendResponse(buildPongNotification(req.GetStreamId())); err != nil {
					return err
				}
				continue
			}

			// Start reading the notifications
			wg.Add(1)
			g.Go(func(req Request) func() error {
				return func() error {
					defer wg.Done()

					session, sub, replay, resumeAvailable, lastSeqNo, done := sessionManager.attach(ctx, req)
					defer session.unsubscribe(sub)

					// READY means the agent accepted this request and attached it to a logical stream session.
					// It does not imply the Kubernetes log source has already produced data.
					if err := sendResponse(buildProtocol(req.GetStreamId(), lastSeqNo, cloud.TestWorkflowNotificationType_WORKFLOW_STREAM_READY, "")); err != nil {
						return err
					}
					if !resumeAvailable {
						if err := sendResponse(buildProtocol(req.GetStreamId(), lastSeqNo, cloud.TestWorkflowNotificationType_WORKFLOW_STREAM_RESUME_UNAVAILABLE, "")); err != nil {
							return err
						}
					}
					for _, event := range replay {
						if err := sendResponse(buildNotification(req.GetStreamId(), event.seqNo, event.notification)); err != nil {
							return err
						}
					}
					if done {
						return nil
					}

					heartbeat := time.NewTicker(workflowNotificationHeartbeatInterval)
					defer heartbeat.Stop()
					drainSubscription := func() error {
						for {
							select {
							case event := <-sub.ch:
								if err := sendResponse(buildNotification(req.GetStreamId(), event.seqNo, event.notification)); err != nil {
									return err
								}
							default:
								return nil
							}
						}
					}
					for {
						select {
						case event := <-sub.ch:
							if err := sendResponse(buildNotification(req.GetStreamId(), event.seqNo, event.notification)); err != nil {
								return err
							}
						case <-sub.done:
							return drainSubscription()
						case <-heartbeat.C:
							if err := sendResponse(buildProtocol(req.GetStreamId(), session.currentSeqNo(), cloud.TestWorkflowNotificationType_WORKFLOW_STREAM_HEARTBEAT, "")); err != nil {
								return err
							}
						case <-ctx.Done():
							return ctx.Err()
						}
					}
				}
			}(req))
		}
	})

	return g.Wait()
}

func buildCloudNotification(streamId string, seqNo uint32, notification *testkube.TestWorkflowExecutionNotification) *cloud.TestWorkflowNotificationsResponse {
	response := &cloud.TestWorkflowNotificationsResponse{
		StreamId:  streamId,
		SeqNo:     seqNo,
		Timestamp: notification.Ts.Format(time.RFC3339Nano),
		Ref:       notification.Ref,
	}
	if notification.Result != nil {
		m, _ := json.Marshal(notification.Result)
		response.Type = cloud.TestWorkflowNotificationType_WORKFLOW_STREAM_RESULT
		response.Message = string(m)
	} else if notification.Output != nil {
		m, _ := json.Marshal(notification.Output)
		response.Type = cloud.TestWorkflowNotificationType_WORKFLOW_STREAM_OUTPUT
		response.Message = string(m)
	} else if notification.EventType == "error" {
		response.Type = cloud.TestWorkflowNotificationType_WORKFLOW_STREAM_ERROR
		response.Message = notification.Log
	} else {
		response.Type = cloud.TestWorkflowNotificationType_WORKFLOW_STREAM_LOG
		response.Message = notification.Log
	}
	return response
}

func buildCloudError(streamId string, message string) *cloud.TestWorkflowNotificationsResponse {
	ts := time.Now()
	return &cloud.TestWorkflowNotificationsResponse{
		StreamId:  streamId,
		SeqNo:     0,
		Timestamp: ts.Format(time.RFC3339Nano),
		Type:      cloud.TestWorkflowNotificationType_WORKFLOW_STREAM_ERROR,
		Message:   fmt.Sprintf("%s %s", ts.Format(constants.PreciseTimeFormat), message),
	}
}

func buildCloudProtocol(streamId string, seqNo uint32, notificationType cloud.TestWorkflowNotificationType, message string) *cloud.TestWorkflowNotificationsResponse {
	return &cloud.TestWorkflowNotificationsResponse{
		StreamId:  streamId,
		SeqNo:     seqNo,
		Timestamp: time.Now().Format(time.RFC3339Nano),
		Type:      notificationType,
		Message:   message,
	}
}

func convertCloudResponseToService(response *cloud.TestWorkflowNotificationsResponse) *cloud.TestWorkflowServiceNotificationsResponse {
	return &cloud.TestWorkflowServiceNotificationsResponse{
		StreamId:  response.StreamId,
		SeqNo:     response.SeqNo,
		Timestamp: response.Timestamp,
		Ref:       response.Ref,
		Type:      response.Type,
		Message:   response.Message,
	}
}

func buildServiceCloudNotification(streamId string, seqNo uint32, notification *testkube.TestWorkflowExecutionNotification) *cloud.TestWorkflowServiceNotificationsResponse {
	return convertCloudResponseToService(buildCloudNotification(streamId, seqNo, notification))
}

func buildServiceCloudError(streamId string, message string) *cloud.TestWorkflowServiceNotificationsResponse {
	return convertCloudResponseToService(buildCloudError(streamId, message))
}

func buildServiceCloudProtocol(streamId string, seqNo uint32, notificationType cloud.TestWorkflowNotificationType, message string) *cloud.TestWorkflowServiceNotificationsResponse {
	return convertCloudResponseToService(buildCloudProtocol(streamId, seqNo, notificationType, message))
}

func buildPongNotification(streamId string) *cloud.TestWorkflowNotificationsResponse {
	return &cloud.TestWorkflowNotificationsResponse{StreamId: streamId, SeqNo: 0}
}

func buildParallelStepPongNotification(streamId string) *cloud.TestWorkflowParallelStepNotificationsResponse {
	return &cloud.TestWorkflowParallelStepNotificationsResponse{StreamId: streamId, SeqNo: 0}
}

func buildServicePongNotification(streamId string) *cloud.TestWorkflowServiceNotificationsResponse {
	return &cloud.TestWorkflowServiceNotificationsResponse{StreamId: streamId, SeqNo: 0}
}

func convertCloudResponseToParallelStep(response *cloud.TestWorkflowNotificationsResponse) *cloud.TestWorkflowParallelStepNotificationsResponse {
	return &cloud.TestWorkflowParallelStepNotificationsResponse{
		StreamId:  response.StreamId,
		SeqNo:     response.SeqNo,
		Timestamp: response.Timestamp,
		Ref:       response.Ref,
		Type:      response.Type,
		Message:   response.Message,
	}
}

func buildParallelStepCloudNotification(streamId string, seqNo uint32, notification *testkube.TestWorkflowExecutionNotification) *cloud.TestWorkflowParallelStepNotificationsResponse {
	return convertCloudResponseToParallelStep(buildCloudNotification(streamId, seqNo, notification))
}

func buildParallelStepCloudError(streamId string, message string) *cloud.TestWorkflowParallelStepNotificationsResponse {
	return convertCloudResponseToParallelStep(buildCloudError(streamId, message))
}

func buildParallelStepCloudProtocol(streamId string, seqNo uint32, notificationType cloud.TestWorkflowNotificationType, message string) *cloud.TestWorkflowParallelStepNotificationsResponse {
	return convertCloudResponseToParallelStep(buildCloudProtocol(streamId, seqNo, notificationType, message))
}
