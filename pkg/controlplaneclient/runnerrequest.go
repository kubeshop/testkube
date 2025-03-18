package controlplaneclient

import "github.com/kubeshop/testkube/pkg/cloud"

type runnerRequestData struct {
	data *cloud.RunnerRequest
	send func(response *cloud.RunnerResponse) error
}

func (r *runnerRequestData) SendError(err error) error {
	return r.send(&cloud.RunnerResponse{
		MessageId:     r.data.MessageId,
		EnvironmentId: r.data.EnvironmentId,
		ExecutionId:   r.data.ExecutionId,
		Type:          r.data.Type,
		Response:      &cloud.RunnerResponse_Error{Error: err.Error()},
	})
}

func (r *runnerRequestData) Ping() RunnerRequestOK {
	return &runnerRequestOk{runnerRequestData: *r}
}

func (r *runnerRequestData) Abort() RunnerRequestOK {
	return &runnerRequestOk{runnerRequestData: *r}
}

func (r *runnerRequestData) Resume() RunnerRequestOK {
	return &runnerRequestOk{runnerRequestData: *r}
}

func (r *runnerRequestData) Pause() RunnerRequestOK {
	return &runnerRequestOk{runnerRequestData: *r}
}

func (r *runnerRequestData) Consider() RunnerRequestConsider {
	return &runnerRequestConsider{runnerRequestData: *r}
}

func (r *runnerRequestData) Start() RunnerRequestStart {
	return &runnerRequestStart{runnerRequestData: *r}
}

type RunnerRequestSpecific[T any] interface {
	RunnerRequestData
	Send(response T) error
}

type RunnerRequestConsider interface {
	RunnerRequestData
	Send(response *cloud.RunnerConsiderResponse) error
}

type RunnerRequestStart interface {
	RunnerRequestData
	Token() string
	Send(response *cloud.RunnerStartResponse) error
}

type RunnerRequestOK interface {
	RunnerRequestData
	Send() error
}

type RunnerRequestData interface {
	Type() cloud.RunnerRequestType
	MessageID() string
	EnvironmentID() string
	ExecutionID() string
	SendError(err error) error
}

//go:generate mockgen -destination=./mock_runnerrequest.go -package=controlplaneclient "github.com/kubeshop/testkube/pkg/controlplaneclient" RunnerRequest
type RunnerRequest interface {
	RunnerRequestData
	Ping() RunnerRequestOK
	Abort() RunnerRequestOK
	Resume() RunnerRequestOK
	Pause() RunnerRequestOK
	Consider() RunnerRequestConsider
	Start() RunnerRequestStart
}

func (r *runnerRequestData) Type() cloud.RunnerRequestType {
	return r.data.Type
}

func (r *runnerRequestData) MessageID() string {
	return r.data.MessageId
}

func (r *runnerRequestData) EnvironmentID() string {
	return r.data.EnvironmentId
}

func (r *runnerRequestData) ExecutionID() string {
	return r.data.ExecutionId
}

type runnerRequestOk struct {
	runnerRequestData
}

func (r *runnerRequestOk) Send() error {
	return r.send(&cloud.RunnerResponse{
		MessageId:     r.data.MessageId,
		EnvironmentId: r.data.EnvironmentId,
		ExecutionId:   r.data.ExecutionId,
		Type:          r.data.Type,
	})
}

type runnerRequestConsider struct {
	runnerRequestData
}

func (r *runnerRequestConsider) Send(response *cloud.RunnerConsiderResponse) error {
	return r.send(&cloud.RunnerResponse{
		MessageId:     r.data.MessageId,
		EnvironmentId: r.data.EnvironmentId,
		ExecutionId:   r.data.ExecutionId,
		Type:          r.data.Type,
		Response:      &cloud.RunnerResponse_Consider{Consider: response},
	})
}

type runnerRequestStart struct {
	runnerRequestData
}

func (r *runnerRequestStart) Token() string {
	return r.data.GetRequest().(*cloud.RunnerRequest_Start).Start.Token
}

func (r *runnerRequestStart) Send(response *cloud.RunnerStartResponse) error {
	return r.send(&cloud.RunnerResponse{
		MessageId:     r.data.MessageId,
		EnvironmentId: r.data.EnvironmentId,
		ExecutionId:   r.data.ExecutionId,
		Type:          r.data.Type,
		Response:      &cloud.RunnerResponse_Start{Start: response},
	})
}
