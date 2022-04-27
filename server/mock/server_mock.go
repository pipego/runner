// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/pipego/scheduler/server/proto (interfaces: ServerProtoClient)

// Package mock_proto is a generated GoMock package.
package mock_proto

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	server "github.com/pipego/scheduler/server/proto"
	grpc "google.golang.org/grpc"
)

// MockServerProtoClient is a mock of ServerProtoClient interface.
type MockServerProtoClient struct {
	ctrl     *gomock.Controller
	recorder *MockServerProtoClientMockRecorder
}

// MockServerProtoClientMockRecorder is the mock recorder for MockServerProtoClient.
type MockServerProtoClientMockRecorder struct {
	mock *MockServerProtoClient
}

// NewMockServerProtoClient creates a new mock instance.
func NewMockServerProtoClient(ctrl *gomock.Controller) *MockServerProtoClient {
	mock := &MockServerProtoClient{ctrl: ctrl}
	mock.recorder = &MockServerProtoClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockServerProtoClient) EXPECT() *MockServerProtoClientMockRecorder {
	return m.recorder
}

// SendServer mocks base method.
func (m *MockServerProtoClient) SendServer(arg0 context.Context, arg1 *server.ServerRequest, arg2 ...grpc.CallOption) (*server.ServerReply, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "SendServer", varargs...)
	ret0, _ := ret[0].(*server.ServerReply)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SendServer indicates an expected call of SendServer.
func (mr *MockServerProtoClientMockRecorder) SendServer(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendServer", reflect.TypeOf((*MockServerProtoClient)(nil).SendServer), varargs...)
}
