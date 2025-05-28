package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"

	pb "gorsovet/cmd/proto"
)

func sFile(stream pb.Gkeeper_ProbaFuncClient, fpath string) (err error) {

	if token == "" {
		return errors.New("no token")
	}
	//fname := filepath.Base(fpath)

	file, err := os.Open(fpath)
	if err != nil {
		return err
	}
	defer file.Close()

	buffer := make([]byte, 64*1024) // 64KB chunks

	// // Send first chunk with filename
	// firstChunk := &pb.Chunk{Filename: fname, Token: token, Metadata: metaFlag}
	// n, err := file.Read(buffer)
	// if err != nil && err != io.EOF {
	// 	return err
	// }
	// // firstChunk.Content = buffer[:n]

	// if err := stream.Send(firstChunk); err != nil {
	// 	return err
	// }

	// Send remaining chunks
	//resp := pb.Chunk{}
	for {
		n, err := file.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if err := stream.Send(&pb.Chunk{
			Content: buffer[:n],
		}); err != nil {
			return err
		}

	}
	//	models.Sugar.Debugf("%s", resp.Content)

	return //stream.CloseSend()
}
func sendFile(stream pb.Gkeeper_UploadFileClient, fpath string) error {

	if token == "" {
		return errors.New("no token")
	}
	fname := filepath.Base(fpath)

	file, err := os.Open(fpath)
	if err != nil {
		return err
	}
	defer file.Close()

	buffer := make([]byte, 64*1024) // 64KB chunks

	// Send first chunk with filename
	firstChunk := &pb.Chunk{Filename: fname, Token: token, Metadata: metaFlag}
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return err
	}
	firstChunk.Content = buffer[:n]

	if err := stream.Send(firstChunk); err != nil {
		return err
	}

	// Send remaining chunks
	//resp := pb.Chunk{}
	for {
		n, err := file.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if err := stream.Send(&pb.Chunk{
			Content: buffer[:n],
		}); err != nil {
			return err
		}

		// Receive acknowledgment from server
		_, err = stream.Recv()
		if err != nil {
			return err
		}
		//		resp = *re
	}
	//	models.Sugar.Debugf("%s", resp.Content)

	return stream.CloseSend()
}
