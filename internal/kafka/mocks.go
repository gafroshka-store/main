package kafka

import (
	"context"
	"reflect"

	"github.com/golang/mock/gomock"
	"github.com/segmentio/kafka-go"
)

// MockReaderInterface мок для ReaderInterface
type MockReaderInterface struct {
	ctrl     *gomock.Controller
	recorder *MockReaderInterfaceMockRecorder
}

func NewMockReaderInterface(ctrl *gomock.Controller) *MockReaderInterface {
	mock := &MockReaderInterface{ctrl: ctrl}
	mock.recorder = &MockReaderInterfaceMockRecorder{mock}
	return mock
}

func (m *MockReaderInterface) EXPECT() *MockReaderInterfaceMockRecorder {
	return m.recorder
}

func (m *MockReaderInterface) ReadMessage(ctx context.Context) (kafka.Message, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReadMessage", ctx)
	ret0, _ := ret[0].(kafka.Message)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (m *MockReaderInterface) Close() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Close")
	return ret[0].(error)
}

type MockReaderInterfaceMockRecorder struct {
	mock *MockReaderInterface
}

func (mr *MockReaderInterfaceMockRecorder) ReadMessage(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(
		mr.mock,
		"ReadMessage",
		reflect.TypeOf((*MockReaderInterface)(nil).ReadMessage),
		ctx,
	)
}

func (mr *MockReaderInterfaceMockRecorder) Close() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(
		mr.mock,
		"Close",
		reflect.TypeOf((*MockReaderInterface)(nil).Close),
	)
}

// MockWriterInterface мок для WriterInterface
type MockWriterInterface struct {
	ctrl     *gomock.Controller
	recorder *MockWriterInterfaceMockRecorder
}

func NewMockWriterInterface(ctrl *gomock.Controller) *MockWriterInterface {
	mock := &MockWriterInterface{ctrl: ctrl}
	mock.recorder = &MockWriterInterfaceMockRecorder{mock}
	return mock
}

func (m *MockWriterInterface) EXPECT() *MockWriterInterfaceMockRecorder {
	return m.recorder
}

func (m *MockWriterInterface) WriteMessages(ctx context.Context, msgs ...kafka.Message) error {
	m.ctrl.T.Helper()
	varargs := []interface{}{ctx}
	for _, msg := range msgs {
		varargs = append(varargs, msg)
	}
	ret := m.ctrl.Call(m, "WriteMessages", varargs...)
	return ret[0].(error)
}

func (m *MockWriterInterface) Close() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Close")
	return ret[0].(error)
}

type MockWriterInterfaceMockRecorder struct {
	mock *MockWriterInterface
}

func (mr *MockWriterInterfaceMockRecorder) WriteMessages(ctx interface{}, msgs ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{ctx}, msgs...)
	return mr.mock.ctrl.RecordCallWithMethodType(
		mr.mock,
		"WriteMessages",
		reflect.TypeOf((*MockWriterInterface)(nil).WriteMessages),
		varargs...,
	)
}

func (mr *MockWriterInterfaceMockRecorder) Close() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(
		mr.mock,
		"Close",
		reflect.TypeOf((*MockWriterInterface)(nil).Close),
	)
}
