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

func (gk *GkeeperService) ListObjects(ctx context.Context, req *pb.ListObjectsRequest) (resp *pb.ListObjectsResponse, err error) {
	response := pb.ListObjectsResponse{Success: false, Reply: "Could not get objects list"}

	db, err := dbase.ConnectToDB(ctx, models.DBEndPoint)
	if err != nil {
		models.Sugar.Debugln(err)
		response.Reply = "ConnectToDB error"
		return &response, status.Errorf(codes.FailedPrecondition, "%s %v", response.Reply, err)
	}
	defer db.CloseBase()

	token := req.GetToken()
	username, err := db.GetUserNameByToken(ctx, token)
	if err != nil {
		response.Reply = "bad GetUserNameByToken"
		models.Sugar.Debugln(err)
		return &response, status.Errorf(codes.Unauthenticated, "%s %v", response.Reply, err)

	}

	response.Listing, err = db.GetObjectsList(ctx, username)
	if err != nil {
		response.Reply = "bad GetObjectsList"
		models.Sugar.Debugln(err)
		return &response, status.Errorf(codes.Unimplemented, "%s %v", response.Reply, err)
	}
	response.Success = true
	response.Reply = "OK"

	return &response, err
}
