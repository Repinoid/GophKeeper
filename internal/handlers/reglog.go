// пакет grpc методов
package handlers

import (
	"context"
	"fmt"
	_ "net/http/pprof"
	"strings"

	pb "gorsovet/cmd/proto"
	"gorsovet/internal/dbase"
	"gorsovet/internal/minio"
	"gorsovet/internal/models"
	"gorsovet/internal/privacy"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GkeeperService struct {
	pb.UnimplementedGkeeperServer
}

func (gk *GkeeperService) RegisterUser(ctx context.Context, req *pb.RegisterRequest) (resp *pb.RegisterResponse, err error) {

	response := pb.RegisterResponse{}

	// сначала подсоединяемся к Базе Данных, недоступна - отбой
	db, err := dbase.ConnectToDB(ctx, models.DBEndPoint)
	if err != nil {
		models.Sugar.Debugln(err)
		response.Success = false
		response.Reply = "ConnectToDB error"
		return &response, err
	}
	defer db.CloseBase()

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
		response.Reply = "Did not find created \"" + strings.ToUpper(userName) + "\" user in DB"
		models.Sugar.Debugln(response.Reply)
		return &response, status.Error(codes.Internal, response.Reply)
	}
	// создаём бакет с именем userName но LowerCase
	err = minio.CreateBucket(ctx, models.MinioClient, strings.ToLower(userName))
	if err != nil {
		models.Sugar.Debugln(err)
		response.Success = false
		response.Reply = "Create \"" + strings.ToLower(userName) + "\" bucket error"
		return &response, err
	}

	response.Success = true
	response.UserId = userId
	response.Reply = "User \"" + strings.ToUpper(userName) + "\" created"

	return &response, nil
}

func (gk *GkeeperService) LoginUser(ctx context.Context, req *pb.LoginRequest) (resp *pb.LoginResponse, err error) {

	response := pb.LoginResponse{}

	// сначала подсоединяемся к Базе Данных, недоступна - отбой
	db, err := dbase.ConnectToDB(ctx, models.DBEndPoint)
	if err != nil {
		models.Sugar.Debugln(err)
		response.Success = false
		response.Reply = "ConnectToDB error"
		return &response, err
	}
	defer db.CloseBase()

	userName := req.GetUsername()
	password := req.GetPassword()
	if userName == "" || password == "" {
		response.Success = false
		response.Reply = "Empty username or password"
		return &response, status.Error(codes.InvalidArgument, response.Reply)
	}
	metadata := req.GetMetadata()

	yes := db.CheckUserPassword(ctx, userName, password)
	if !yes {
		response.Success = false
		response.Reply = "Wrong username/password"
		models.Sugar.Debugln(response.Reply)
		return &response, status.Error(codes.AlreadyExists, response.Reply)
	}
	Token, err := privacy.BuildJWTString(userName, models.JWTKey)
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}
	err = db.PutToken(ctx, userName, Token, metadata)
	if err != nil {
		models.Sugar.Debugln(err)
		response.Success = false
		response.Reply = "PutToken error"
		return &response, err
	}

	response.Success = true
	response.Token = Token
	response.Reply = "Auth OK"

	return &response, nil
}
