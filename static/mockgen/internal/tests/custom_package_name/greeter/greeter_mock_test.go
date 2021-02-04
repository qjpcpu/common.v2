// Code generated by MockGen. DO NOT EDIT.
// Source: greeter.go

// Package greeter is a generated GoMock package.
package greeter

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	client "github.com/qjpcpu/common.v2/static/mockgen/internal/tests/custom_package_name/client/v1"
)

// MockInputMaker is a mock of InputMaker interface.
type MockInputMaker struct {
	ctrl     *gomock.Controller
	recorder *MockInputMakerMockRecorder
}

// MockInputMakerMockRecorder is the mock recorder for MockInputMaker.
type MockInputMakerMockRecorder struct {
	mock *MockInputMaker
}

// NewMockInputMaker creates a new mock instance.
func NewMockInputMaker(ctrl *gomock.Controller) *MockInputMaker {
	mock := &MockInputMaker{ctrl: ctrl}
	mock.recorder = &MockInputMakerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockInputMaker) EXPECT() *MockInputMakerMockRecorder {
	return m.recorder
}

// MakeInput mocks base method.
func (m *MockInputMaker) MakeInput() client.GreetInput {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "MakeInput")
	ret0, _ := ret[0].(client.GreetInput)
	return ret0
}

// MakeInput indicates an expected call of MakeInput.
func (mr *MockInputMakerMockRecorder) MakeInput() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MakeInput", reflect.TypeOf((*MockInputMaker)(nil).MakeInput))
}
