package handlers

import (
	"bytes"
	"encoding/hex"
	"gorsovet/internal/dbase"
	"gorsovet/internal/minios3"
	"gorsovet/internal/models"
	"gorsovet/internal/privacy"
	"io"
	"strconv"
	"time"

	pb "gorsovet/cmd/proto"

	"github.com/minio/minio-go/v7/pkg/encrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Gsender - send данных из сервера в клиент
func (gk *GkeeperService) Gsender(req *pb.SenderRequest, stream pb.Gkeeper_GsenderServer) (err error) {

	ctx := stream.Context()

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

	// Конвертируем в time.Time
	t := param.GetCreatedAt().AsTime()

	md := metadata.Pairs(
		"FileName", param.GetFileurl(),
		"MetaData", param.GetMetadata(),
		"DataType", param.DataType,
		"Size", strconv.Itoa(int(param.GetSize())),
		"CreatedAt", t.Format(time.RFC3339),
	)
	// Set then Send md
	// Send without Set - does not work
	if err := stream.SetHeader(md); err != nil {
		return err
	}
	if err := stream.SendHeader(md); err != nil {
		return err
	}

	// Create a buffer to hold chunks
	buffer := make([]byte, 64*1024) // 64KB chunks

	reader := bytes.NewReader(fileBytes)

	// // Send first chunk with filename etc
	// firstChunk := &pb.SenderChunk{Filename: param.GetFileurl(), Metadata: param.GetMetadata(),
	// 	DataType: param.DataType, Size: param.GetSize(), CreatedAt: param.GetCreatedAt()}
	// n, err := reader.Read(buffer)
	// if err != nil && err != io.EOF {
	// 	return
	// }
	// firstChunk.Content = buffer[:n]
	// if err = stream.Send(firstChunk); err != nil {
	// 	return
	// }

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
