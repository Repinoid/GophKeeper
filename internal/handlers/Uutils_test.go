package handlers

import (
	"context"
	pb "gorsovet/cmd/proto"
	"io"

	"google.golang.org/grpc/metadata"
)

// MockClientStream обманка stream pb.Gkeeper_GreceiverServer
//
//	type Gkeeper_GreceiverServer interface {
//			SendAndClose(*ReceiverResponse) error
//			Recv() (*ReceiverChunk, error)
//			grpc.ServerStream - методы RecvMsg, Context, SendHeader, SendMsg, SetHeader, SetTrailer
//	}
type MockClientStream struct {
	Ctx         context.Context
	recvMsgs    []*pb.ReceiverChunk
	currentRecv int
}

func (m *MockClientStream) Context() context.Context {
	return m.Ctx
}
func (m *MockClientStream) Recv() (a *pb.ReceiverChunk, err error) {
	if m.currentRecv >= len(m.recvMsgs) {
		return nil, io.EOF
	}
	msg := m.recvMsgs[m.currentRecv]
	m.currentRecv++
	return msg, nil
}
func (m *MockClientStream) RecvMsg(msg interface{}) error {
	return nil
}
func (m *MockClientStream) SendAndClose(a *pb.ReceiverResponse) error {
	return nil
}
func (m *MockClientStream) SendHeader(metadata.MD) error {
	return nil
}
func (m *MockClientStream) SendMsg(msg interface{}) error {
	return nil
}
func (m *MockClientStream) SetHeader(metadata.MD) error {
	return nil
}
func (m *MockClientStream) SetTrailer(metadata.MD) {
}
