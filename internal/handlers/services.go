package handlers

import (
	"context"
	_ "net/http/pprof"
	"strings"

	pb "gorsovet/cmd/proto"
	"gorsovet/internal/dbase"
	"gorsovet/internal/minio"
	"gorsovet/internal/models"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GkeeperService struct {
	pb.UnimplementedGkeeperServer
}

func (gk *GkeeperService) RegisterUser(ctx context.Context, req *pb.RegisterRequest) (resp *pb.RegisterResponse, err error) {

	var response pb.RegisterResponse

	// сначала подсоединяемся к Базе Данных, недоступна - отбой
	db, err := dbase.ConnectToDB(ctx, models.DBEndPoint)
	if err != nil {
		models.Sugar.Debugln(err)
		response.Success = false
		response.Reply = "ConnectToDB error"
		return &response, err
	}
	// S3
	minioClient, err := minio.ConnectToS3()
	if err != nil {
		models.Sugar.Debugln(err)
		response.Success = false
		response.Reply = "S3 connection error"
		return &response, err
	}

	userName := req.GetUsername()
	password := req.GetPassword()
	if userName == "" || password == "" {
		response.Success = false
		response.Reply = "Empty username or password"
		return &response, status.Error(codes.InvalidArgument, response.Reply)
	}
	metadata := req.GetMetadata()

	yes, _ := db.IfUserExists(ctx, userName)
	if yes {
		response.Success = false
		response.Reply = "User \"" + strings.ToUpper(userName) + "\" already exists"
		models.Sugar.Debugln(response.Reply)
		return &response, status.Error(codes.AlreadyExists, response.Reply)
	}

	err = db.AddUser(ctx, userName, password, metadata)
	if err != nil {
		models.Sugar.Debugln(err)
		response.Success = false
		response.Reply = "AddUser error"
		return &response, err
	}
	// получение userId, заодно удостоверяемся что регистрация прошла успешно
	yes, userId := db.IfUserExists(ctx, userName)
	if !yes {
		response.Success = false
		response.Reply = "Did not find user in DB"
		models.Sugar.Debugln(response.Reply)
		return &response, status.Error(codes.Internal, response.Reply)
	}

	err = minio.CreateBucket(ctx, minioClient, strings.ToLower(userName))

	response.Success = true
	response.UserId = userId
	response.Reply = "OK"

	return &response, nil
}
