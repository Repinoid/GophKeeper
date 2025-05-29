package handlers

import (
	"context"
	_ "net/http/pprof"

	pb "gorsovet/cmd/proto"
	"gorsovet/internal/dbase"
	"gorsovet/internal/minio"
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

func (gk *GkeeperService) RemoveObjects(ctx context.Context, req *pb.RemoveObjectsRequest) (resp *pb.RemoveObjectsResponse, err error) {
	response := pb.RemoveObjectsResponse{Success: false, Reply: "Could not remove objects"}
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

	// удалить запись в базе данных, заодно получить имя файла для удаления в S3
	fnam, err := db.RemoveObjects(ctx, username, req.GetObjectId())
	if err != nil {
		response.Reply = "bad RemoveObjects"
		models.Sugar.Debugln(err)
		return &response, status.Errorf(codes.Unimplemented, "%s %v", response.Reply, err)
	}
	// получить имя бакета, может быть иным чем юзернейм
	_, bucketname, err := db.GetBucketKeyByUserName(ctx, username)
	if err != nil {
		response.Reply = "bad GetBucketKeyByUserName "
		models.Sugar.Debugln(err)
		return &response, status.Errorf(codes.Unimplemented, "%s %v", response.Reply, err)
	}
	// удалить файл в бакете
	err = minio.S3RemoveFile(ctx, models.MinioClient, bucketname, fnam)
	if err != nil {
		response.Reply = "bad S3RemoveFile"
		models.Sugar.Debugln(err)
		return &response, status.Errorf(codes.Unimplemented, "%s %v", response.Reply, err)
	}
	response.Success = true
	response.Reply = "OK remove object"

	return &response, err

}
