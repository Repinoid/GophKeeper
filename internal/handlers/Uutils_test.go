package handlers

import (
	"bytes"
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
	// без этого трюка, просто с metadata.NewOutgoingContext метаданные 
	// прописываются только куда то вглубь, FromIncomingContext на сервере их не видит
	inMd, _ := metadata.FromOutgoingContext(m.Ctx)
	ctx := metadata.NewIncomingContext(m.Ctx, inMd)
	return ctx
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

func makeMockStream(ctx context.Context, data []byte) (mockStream *MockClientStream, err error) {

	reader := bytes.NewReader(data)
	buffer := make([]byte, 64*1024) // 64KB chunks

	// Send first chunk with filename
	msgs := []*pb.ReceiverChunk{}

	// Send remaining chunks
	for {
		n, err := reader.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		chunk := pb.ReceiverChunk{Content: buffer[:n]}
		msgs = append(msgs, &chunk)
	}
	mockS := &MockClientStream{
		Ctx:      ctx,
		recvMsgs: msgs,
	}
	mockStream = mockS
	return
}
