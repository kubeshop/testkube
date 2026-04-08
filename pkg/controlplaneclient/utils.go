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
)

type notificationStreamEvent struct {
	seqNo        uint32
	notification *testkube.TestWorkflowExecutionNotification
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
	nextSeqNo   uint32
	replay      []notificationStreamEvent
	replayBytes int
	subscribers map[uint64]*notificationStreamSubscription
	done        bool
	lastSeqNo   uint32
	lastActive  time.Time
}

func newNotificationStreamSession() *notificationStreamSession {
	return &notificationStreamSession{
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
	s.mu.Lock()
	seqNo := s.nextSeqNo
	s.nextSeqNo++
	s.lastSeqNo = seqNo
	s.lastActive = time.Now()

	event := notificationStreamEvent{
		seqNo:        seqNo,
		notification: notification,
	}
	s.replay = append(s.replay, event)
	s.replayBytes += approximateNotificationBytes(notification)
	for len(s.replay) > workflowNotificationReplayMaxEvents || s.replayBytes > workflowNotificationReplayMaxBytes {
		s.replayBytes -= approximateNotificationBytes(s.replay[0].notification)
		s.replay[0].notification = nil
		s.replay = s.replay[1:]
	}

	for _, sub := range s.subscribers {
		subscribers = append(subscribers, sub)
	}
	s.mu.Unlock()

	for _, sub := range subscribers {
		sub.send(event)
	}
}

func (s *notificationStreamSession) close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.done {
		return
	}
	s.done = true
	s.lastActive = time.Now()
	for id, sub := range s.subscribers {
		sub.close()
		delete(s.subscribers, id)
	}
}

func (s *notificationStreamSession) currentSeqNo() uint32 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastSeqNo
}

func (s *notificationStreamSession) isDone() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.done
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

func (s *notificationStreamSession) expired(now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.done && now.Sub(s.lastActive) >= workflowNotificationSessionIdleTTL
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
	mu       sync.Mutex
	nextID   atomic.Uint64
	sessions map[string]*notificationStreamSession
	key      func(Request) string
	process  func(ctx context.Context, req Request) NotificationWatcher
}

func newNotificationStreamSessionManager[Request notificationRequest](
	key func(Request) string,
	process func(ctx context.Context, req Request) NotificationWatcher,
) *notificationStreamSessionManager[Request] {
	return &notificationStreamSessionManager[Request]{
		sessions: make(map[string]*notificationStreamSession),
		key:      key,
		process:  process,
	}
}

func (m *notificationStreamSessionManager[Request]) attach(ctx context.Context, req Request) (*notificationStreamSession, *notificationStreamSubscription, []notificationStreamEvent, bool, uint32, bool) {
	key := m.key(req)
	now := time.Now()

	m.mu.Lock()
	for sessionKey, session := range m.sessions {
		if session.expired(now) {
			delete(m.sessions, sessionKey)
		}
	}

	session := m.sessions[key]
	if session != nil && session.isDone() && req.GetResumeAfterSeqNo() == 0 {
		delete(m.sessions, key)
		session = nil
	}
	if session == nil {
		session = newNotificationStreamSession()
		m.sessions[key] = session
		go m.runSource(ctx, session, req)
	}
	subscriptionID := m.nextID.Add(1)
	m.mu.Unlock()

	sub, replay, available, lastSeqNo, done := session.subscribe(req.GetResumeAfterSeqNo(), subscriptionID)
	return session, sub, replay, available, lastSeqNo, done
}

func (m *notificationStreamSessionManager[Request]) runSource(ctx context.Context, session *notificationStreamSession, req Request) {
	defer session.close()

	watcher := m.process(ctx, req)
	for notification := range watcher.Channel() {
		session.publish(notification)
	}
	if watcher.Err() != nil {
		session.publish(&testkube.TestWorkflowExecutionNotification{
			Ts:        time.Now(),
			EventType: "error",
			Log:       fmt.Sprintf("%s %s", time.Now().Format(constants.PreciseTimeFormat), watcher.Err().Error()),
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
	sessionManager := newNotificationStreamSessionManager(sessionKey, process)

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
				logger.Errorw("process notifications error", "error", err)
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
					logger.Errorw("process notifications error", "error", err)
					return err
				}
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(recvTimeout):
				err = errors.New("receive request too slow")
				logger.Errorw("process notifications error", "error", err)
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
