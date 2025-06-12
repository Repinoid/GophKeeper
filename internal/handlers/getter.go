package handlers

import (
	"context"
	_ "net/http/pprof"
	"time"

	pb "gorsovet/cmd/proto"
	"gorsovet/internal/dbase"
	"gorsovet/internal/minios3"

	//"gorsovet/internal/minio"
	"gorsovet/internal/models"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (gk *GkeeperService) Gping(ctx context.Context, req *pb.PingRequest) (*pb.PingResponse, error) {
	return &pb.PingResponse{
		Status:    "OK",
		Timestamp: time.Now().Unix(),
	}, nil
}

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

// RemoveObjects - удаление объекта
func (gk *GkeeperService) RemoveObjects(ctx context.Context, req *pb.RemoveObjectsRequest) (resp *pb.RemoveObjectsResponse, err error) {
	// по умолчанию - неудача, прописываем это в response
	response := pb.RemoveObjectsResponse{Success: false, Reply: "Could not remove objects"}
	db, err := dbase.ConnectToDB(ctx, models.DBEndPoint)
	if err != nil {
		models.Sugar.Debugln(err)
		response.Reply = "ConnectToDB error"
		return &response, status.Errorf(codes.FailedPrecondition, "%s %v", response.Reply, err)
	}
	defer db.CloseBase()
	// токен передан в req(uest)
	token := req.GetToken()

	// GetUserNameByToken получаем имя юзера по токену (из таблицы TOKENA)
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
	// получить имя бакета, может быть иным чем юзернейм, GetBucketKeyByUserName возвращает ключ шифрования и имя бакета, ключ здесь не нужен
	_, bucketname, err := db.GetBucketKeyByUserName(ctx, username)
	if err != nil {
		response.Reply = "bad GetBucketKeyByUserName "
		models.Sugar.Debugln(err)
		return &response, status.Errorf(codes.Unimplemented, "%s %v", response.Reply, err)
	}
	// удалить файл в бакете
	err = minios3.S3RemoveFile(ctx, models.MinioClient, bucketname, fnam)
	if err != nil {
		response.Reply = "bad S3RemoveFile"
		models.Sugar.Debugln(err)
		return &response, status.Errorf(codes.Unimplemented, "%s %v", response.Reply, err)
	}
	// если добрались до этой строчки, значит Ок, прописываем его в response и возвращаем
	response.Success = true
	response.Reply = "OK remove object"

	return &response, nil

}
