// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/kubeshop/testkube/pkg/credentials (interfaces: CredentialRepository)

// Package credentials is a generated GoMock package.
package credentials

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockCredentialRepository is a mock of CredentialRepository interface.
type MockCredentialRepository struct {
	ctrl     *gomock.Controller
	recorder *MockCredentialRepositoryMockRecorder
}

// MockCredentialRepositoryMockRecorder is the mock recorder for MockCredentialRepository.
type MockCredentialRepositoryMockRecorder struct {
	mock *MockCredentialRepository
}

// NewMockCredentialRepository creates a new mock instance.
func NewMockCredentialRepository(ctrl *gomock.Controller) *MockCredentialRepository {
	mock := &MockCredentialRepository{ctrl: ctrl}
	mock.recorder = &MockCredentialRepositoryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCredentialRepository) EXPECT() *MockCredentialRepositoryMockRecorder {
	return m.recorder
}

// Get mocks base method.
func (m *MockCredentialRepository) Get(arg0 context.Context, arg1 string) ([]byte, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", arg0, arg1)
	ret0, _ := ret[0].([]byte)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *MockCredentialRepositoryMockRecorder) Get(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockCredentialRepository)(nil).Get), arg0, arg1)
}
