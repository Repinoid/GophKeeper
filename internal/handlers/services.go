package handlers

import (
	"context"
	_ "net/http/pprof"

	pb "gorsovet/cmd/proto"
	"gorsovet/internal/dbase"
	"gorsovet/internal/models"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GkeeperService struct {
	pb.UnimplementedGkeeperServer
}

func (gk *GkeeperService) RegisterUser(ctx context.Context, req *pb.RegisterRequest) (resp *pb.RegisterResponse, err error) {

	var response pb.RegisterResponse

	userName := req.GetUsername()
	password := req.GetPassword()
	if userName == "" || password == "" {
		response.Success = false
		response.Reply = "Empty username or password"
		return &response, status.Error(codes.InvalidArgument, response.Reply)
	}
	metadata := req.GetMetadata()

	db, err := dbase.ConnectToDB(ctx, models.DBEndPoint)
	if err != nil {
		models.Sugar.Debugln(err)
		response.Success = false
		response.Reply = "ConnectToDB error"
		return &response, err
	}
	err = db.AddUser(ctx, userName, password, metadata)
	if err != nil {
		models.Sugar.Debugln(err)
		response.Success = false
		response.Reply = "AddUser error"
		return &response, err
	}
	yes, userId := db.IfUserExists(ctx, userName)
	if !yes {
		response.Success = false
		response.Reply = "Did not find user in DB"
		return &response, status.Error(codes.Internal, response.Reply)
	}
	response.Success = true
	response.UserId = userId
	response.Reply = "OK"

	return &response, nil
}
