package main

import (
	"context"
	"errors"
	"fmt"
	"io"

	pb "gorsovet/cmd/proto"
	"gorsovet/internal/models"
)

func AddUser(ctx context.Context, client pb.GkeeperClient, username, password string) (err error) {
	req := &pb.RegisterRequest{Username: username, Password: password, Metadata: metaFlag}
	resp, err := client.RegisterUser(ctx, req)
	if err != nil {
		return
	}
	models.Sugar.Debugf("%+v", resp)
	return
}

func Login(ctx context.Context, client pb.GkeeperClient, username, password string) (token string, err error) {
	req := &pb.LoginRequest{Username: username, Password: password, Metadata: metaFlag}
	resp, err := client.LoginUser(ctx, req)
	if err != nil {
		return
	}
	token = resp.GetToken()
	if token == "" {
		return "", errors.New("login did not return token")
	}
	models.Sugar.Debugf("%+v", resp.Reply)
	return
}

func GetList(ctx context.Context, client pb.GkeeperClient) (list []*pb.ObjectParams, err error) {
	if token == "" {
		return nil, errors.New("no token")
	}
	reqList := &pb.ListObjectsRequest{Token: token}
	resp, err := client.ListObjects(ctx, reqList)
	if err != nil {
		models.Sugar.Debugf("No listing %v\n", err)
		fmt.Printf("No listing %v\n", err)
		return
	}
	list = resp.GetListing()
	return
}

// IfIdExist проверяем существует ли запись с номером id
func IfIdExist(ctx context.Context, client pb.GkeeperClient, id int32) bool {
	list, err := GetList(ctx, client)
	if err != nil {
		return false
	}
	for _, value := range list {
		if id == value.GetId() {
			return true
		}
	}
	return false
}

// Remover удаляем запись (ака объект) с номером id
func Remover(ctx context.Context, client pb.GkeeperClient, id int) (err error) {
	if token == "" {
		return errors.New("no token")
	}

	req := &pb.RemoveObjectsRequest{ObjectId: int32(id), Token: token}
	resp, err := client.RemoveObjects(ctx, req)
	if err != nil {
		models.Sugar.Debugf("No listing %v\n", err)
		fmt.Printf("No listing %v\n", err)
		return
	}
	if !resp.Success {
		return fmt.Errorf("could not delete object number %d", id)
	}

	return
}

// receiveFile - получаем запись из хранилища
func receiveFile(stream pb.Gkeeper_GsenderClient) (chuvak *pb.SenderChunk, err error) {
	if token == "" {
		return nil, errors.New("no token")
	}
	// в первом куске помимо данных - параметры записи
	chu := pb.SenderChunk{}
	firstChunk, err := stream.Recv()
	if err != nil {
		models.Sugar.Debugf("stream.Recv()  %v", err)
		return nil, err
	}
	chu.Content = firstChunk.GetContent()
	chu.Filename = firstChunk.GetFilename()
	chu.Metadata = firstChunk.GetMetadata()
	chu.Size = firstChunk.GetSize()
	chu.CreatedAt = firstChunk.GetCreatedAt()
	chu.DataType = firstChunk.GetDataType()

	// Process subsequent chunks
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		chu.Content = append(chu.Content, chunk.GetContent()...)
	}
	return &chu, err
}
