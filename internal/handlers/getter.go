package handlers

import (
	"bytes"
	"context"
	"encoding/hex"
	"io"
	_ "net/http/pprof"

	pb "gorsovet/cmd/proto"
	"gorsovet/internal/dbase"
	"gorsovet/internal/minios3"

	//"gorsovet/internal/minio"
	"gorsovet/internal/models"
	"gorsovet/internal/privacy"

	"github.com/minio/minio-go/v7/pkg/encrypt"
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

// Gsender - send данных из сервера в клиент
func (gk *GkeeperService) Gsender(req *pb.SenderRequest, stream pb.Gkeeper_GsenderServer) (err error) {
	ctx := context.Background()
	//response := pb.RemoveObjectsResponse{Success: false, Reply: "Could not get objects"}

	db, err := dbase.ConnectToDB(ctx, models.DBEndPoint)
	if err != nil {
		models.Sugar.Debugln(err)
		//	response.Reply = "ConnectToDB error"
		return status.Errorf(codes.FailedPrecondition, "%s %v", "ConnectToDB error", err)
		//return status.Errorf(codes.FailedPrecondition, "%s %v", response.Reply, err)
	}
	defer db.CloseBase()

	token := req.GetToken()
	username, err := db.GetUserNameByToken(ctx, token)
	if err != nil {
		//	response.Reply = "bad GetUserNameByToken"
		models.Sugar.Debugln(err)
		return status.Errorf(codes.Unauthenticated, "%s %v", "bad GetUserNameByToken", err)
	}
	object_id := req.GetObjectId()
	param, err := db.GetObjectIdParams(ctx, username, object_id)

	// в param.Filekey ключ файла в HEX, переводим в байты
	codedObjectKey, err := hex.DecodeString(param.GetFilekey())
	if err != nil {
		return
	}
	// получаем ключ бакета из таблицы юзеров
	bucketKeyHex, bucketName, err := db.GetBucketKeyByUserName(ctx, username)
	if err != nil {
		models.Sugar.Debugf("GetBucketKeyByUserName  %v", err)
		return
	}
	// в bucketKeyHex - ключ бакета, шифрованный мастер-ключом.  переводим его сначала из HEX в байты
	codedBucketkey, err := hex.DecodeString(bucketKeyHex)
	if err != nil {
		models.Sugar.Debugf("hex.DecodeString  %v", err)
		return
	}
	// deкодируем ключ бакета мастер-ключом
	bucketKey, err := privacy.DecryptB2B(codedBucketkey, models.MasterKey)
	if err != nil {
		models.Sugar.Debugf("privacy.DecryptB2B  %v", err)
		return
	}
	// раскодируем ключ файла при помощи уже раскодированного ключа бакета
	objectKey, err := privacy.DecryptB2B(codedObjectKey, bucketKey)
	// sse - криптоключ для шифрования файла при записи/read Minio
	sse, err := encrypt.NewSSEC(objectKey)
	if err != nil {
		return
	}
	// читаем файл из бакета
	fileBytes, err := minios3.S3GetFileBytes(ctx, models.MinioClient, bucketName, param.Fileurl, sse)
	if err != nil {
		models.Sugar.Debugf("minio.S3GetFileBytes  %v", err)
		return
	}

	// Create a buffer to hold chunks
	buffer := make([]byte, 64*1024) // 64KB chunks

	reader := bytes.NewReader(fileBytes)

	// Send first chunk with filename etc
	firstChunk := &pb.SenderChunk{Filename: param.GetFileurl(), Metadata: param.GetMetadata(),
		DataType: param.DataType, Size: param.GetSize(), CreatedAt: param.GetCreatedAt()}
	n, err := reader.Read(buffer)
	if err != nil && err != io.EOF {
		return
	}
	firstChunk.Content = buffer[:n]
	if err = stream.Send(firstChunk); err != nil {
		return
	}

	for {
		bytesRead, err := reader.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		// Send chunk to client
		if err := stream.Send(&pb.SenderChunk{
			Content: buffer[:bytesRead],
		}); err != nil {
			return err
		}
	}
	return
}
