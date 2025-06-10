package handlers

import (
	"context"
	pb "gorsovet/cmd/proto"
	"io"

	"google.golang.org/grpc/metadata"
)

// type Gkeeper_GsenderServer interface {
// 	Send(*SenderChunk) error
// 	grpc.ServerStream
// }

// MockStream implements the server-side stream interface
type MockSendStream struct {
	//	grpc.ServerStream
	MessagesToSend []*pb.SenderChunk
	Received       *pb.SenderRequest
	Ctx            context.Context
}

func (m *MockSendStream) Send(resp *pb.SenderChunk) error {
	m.MessagesToSend = append(m.MessagesToSend, resp)
	return nil
}

func (m *MockSendStream) Recv() (*pb.SenderChunk, error) {
	if len(m.MessagesToSend) == 0 {
		return nil, io.EOF
	}
	msg := m.MessagesToSend[0]
	m.MessagesToSend = m.MessagesToSend[1:]
	return msg, nil
}

func (m *MockSendStream) Context() context.Context {
	if m.Ctx != nil {
		return m.Ctx
	}
	return context.Background()
}
func (m *MockSendStream) RecvMsg(msg any) error {
	return nil
}
func (m *MockSendStream) SendHeader(metadata.MD) error {
	return nil
}
func (m *MockSendStream) SendMsg(msg any) error {
	return nil
}
func (m *MockSendStream) SetHeader(metadata.MD) error {
	return nil
}
func (m *MockSendStream) SetTrailer(metadata.MD) {
}

func (suite *TstHand) Test09Gsender() {

	server := suite.serv

	req := &pb.SenderRequest{ObjectId: 2, Token: suite.token}

	mockStream := &MockSendStream{}

	err := server.Gsender(req, mockStream)
	suite.Require().NoError(err)

	a := mockStream.MessagesToSend
	_ = a

}
