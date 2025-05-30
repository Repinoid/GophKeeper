package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	pb "gorsovet/cmd/proto"
)

func sendFile(stream pb.Gkeeper_GreceiverClient, fpath string) (resp *pb.ReceiverResponse, err error) {

	if token == "" {
		return nil, errors.New("no token")
	}
	fname := filepath.Base(fpath)

	file, err := os.Open(fpath)
	if err != nil {
		return
	}
	defer file.Close()

	buffer := make([]byte, 64*1024) // 64KB chunks

	// Send first chunk with filename etc
	firstChunk := &pb.ReceiverChunk{Filename: fname, Token: token, Metadata: metaFlag, DataType: "file", ObjectId: int32(updateFlag)}
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return
	}
	firstChunk.Content = buffer[:n]

	if err = stream.Send(firstChunk); err != nil {
		return
	}
	// Send remaining chunks
	for {
		n, err := file.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if err := stream.Send(&pb.ReceiverChunk{
			Content: buffer[:n],
		}); err != nil {
			return nil, err
		}
	}
	resp, err = stream.CloseAndRecv()
	return
}

func sendText(stream pb.Gkeeper_GreceiverClient, text, objectName string, dtype string) (resp *pb.ReceiverResponse, err error) {

	if token == "" {
		return nil, errors.New("no token")
	}
	
	reader := strings.NewReader(text)

	buffer := make([]byte, 64*1024) // 64KB chunks

	// Send first chunk with filename
	firstChunk := &pb.ReceiverChunk{Filename: objectName, Token: token, Metadata: metaFlag, DataType: dtype, ObjectId: int32(updateFlag)}
	n, err := reader.Read(buffer)
	if err != nil && err != io.EOF {
		return
	}
	firstChunk.Content = buffer[:n]

	if err = stream.Send(firstChunk); err != nil {
		return
	}
	// Send remaining chunks
	for {
		n, err := reader.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if err := stream.Send(&pb.ReceiverChunk{
			Content: buffer[:n],
		}); err != nil {
			return nil, err
		}
	}
	resp, err = stream.CloseAndRecv()
	return
}
