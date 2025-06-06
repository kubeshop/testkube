// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/kubeshop/testkube/pkg/repository/sequence (interfaces: Repository)

// Package sequence is a generated GoMock package.
package sequence

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockRepository is a mock of Repository interface.
type MockRepository struct {
	ctrl     *gomock.Controller
	recorder *MockRepositoryMockRecorder
}

// MockRepositoryMockRecorder is the mock recorder for MockRepository.
type MockRepositoryMockRecorder struct {
	mock *MockRepository
}

// NewMockRepository creates a new mock instance.
func NewMockRepository(ctrl *gomock.Controller) *MockRepository {
	mock := &MockRepository{ctrl: ctrl}
	mock.recorder = &MockRepositoryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockRepository) EXPECT() *MockRepositoryMockRecorder {
	return m.recorder
}

// DeleteAllExecutionNumbers mocks base method.
func (m *MockRepository) DeleteAllExecutionNumbers(arg0 context.Context, arg1 ExecutionType) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteAllExecutionNumbers", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteAllExecutionNumbers indicates an expected call of DeleteAllExecutionNumbers.
func (mr *MockRepositoryMockRecorder) DeleteAllExecutionNumbers(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteAllExecutionNumbers", reflect.TypeOf((*MockRepository)(nil).DeleteAllExecutionNumbers), arg0, arg1)
}

// DeleteExecutionNumber mocks base method.
func (m *MockRepository) DeleteExecutionNumber(arg0 context.Context, arg1 string, arg2 ExecutionType) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteExecutionNumber", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteExecutionNumber indicates an expected call of DeleteExecutionNumber.
func (mr *MockRepositoryMockRecorder) DeleteExecutionNumber(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteExecutionNumber", reflect.TypeOf((*MockRepository)(nil).DeleteExecutionNumber), arg0, arg1, arg2)
}

// DeleteExecutionNumbers mocks base method.
func (m *MockRepository) DeleteExecutionNumbers(arg0 context.Context, arg1 []string, arg2 ExecutionType) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteExecutionNumbers", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteExecutionNumbers indicates an expected call of DeleteExecutionNumbers.
func (mr *MockRepositoryMockRecorder) DeleteExecutionNumbers(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteExecutionNumbers", reflect.TypeOf((*MockRepository)(nil).DeleteExecutionNumbers), arg0, arg1, arg2)
}

// GetNextExecutionNumber mocks base method.
func (m *MockRepository) GetNextExecutionNumber(arg0 context.Context, arg1 string, arg2 ExecutionType) (int32, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetNextExecutionNumber", arg0, arg1, arg2)
	ret0, _ := ret[0].(int32)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetNextExecutionNumber indicates an expected call of GetNextExecutionNumber.
func (mr *MockRepositoryMockRecorder) GetNextExecutionNumber(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetNextExecutionNumber", reflect.TypeOf((*MockRepository)(nil).GetNextExecutionNumber), arg0, arg1, arg2)
}
