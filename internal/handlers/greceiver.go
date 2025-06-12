package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	_ "net/http/pprof"
	"strconv"

	pb "gorsovet/cmd/proto"
	"gorsovet/internal/dbase"
	"gorsovet/internal/minios3"
	"gorsovet/internal/models"
	"gorsovet/internal/privacy"

	"github.com/minio/minio-go/v7/pkg/encrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func (gk *GkeeperService) Greceiver(stream pb.Gkeeper_GreceiverServer) (err error) {

	var fname, dataType, token, metaData string
	var object_id int32

	// считываем параметры потока из заголовка
	ctx := stream.Context()
//	md, ok := metadata.FromIncomingContext(stream.Context())
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		values := md.Get("Token")
		if len(values) == 0 {
			models.Sugar.Debugf("no Token")
			return status.Error(codes.NotFound, "нет Token")
		}
		token = values[0]

		values = md.Get("FileName")
		if len(values) == 0 {
			models.Sugar.Debugf("no filename")
			return status.Error(codes.NotFound, "нет filename")
		}
		fname = values[0]

		values = md.Get("DataType")
		if len(values) == 0 {
			models.Sugar.Debugf("no DataType")
			return status.Error(codes.NotFound, "нет Datatype")
		}
		dataType = values[0]

		values = md.Get("MetaData")
		if len(values) == 0 {
			models.Sugar.Debugf("no MetaData")
			return status.Error(codes.NotFound, "нет MetaData")
		}
		metaData = values[0]

		values = md.Get("ObjectId")
		if len(values) == 0 {
			models.Sugar.Debugf("no ObjectId")
			return status.Error(codes.NotFound, "нет ObjectId")
		}
		object_id0, _ := strconv.Atoi(values[0])
		object_id = int32(object_id0)
	}

	fileContent := []byte{}
	// вычитываем контент посылки
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		fileContent = append(fileContent, chunk.GetContent()...)
	}

	// затем подсоединяемся к Базе Данных, недоступна - отбой
	db, err := dbase.ConnectToDB(ctx, models.DBEndPoint)
	if err != nil {
		models.Sugar.Debugf("ConnectToDB  %v", err)
		return
	}
	defer db.CloseBase()

	//token = firstChunk.GetToken()
	userName, err := db.GetUserNameByToken(ctx, token)
	if err != nil {
		models.Sugar.Debugf("GetUserNameByToken  %v", err)
		return
	}

	bucketKeyHex, bucketName, err := db.GetBucketKeyByUserName(ctx, userName)
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
	//metaData = firstChunk.GetMetadata()

	// создаём случайный ключ для шифрования файла
	fileKey := make([]byte, 32)
	_, err = rand.Read(fileKey)
	if err != nil {
		return
	}
	// NewSSEC returns a new server-side-encryption using SSE-C and the provided key. The key must be 32 bytes long
	// sse - криптоключ для шифрования файла при записи в Minio
	// Requests specifying Server Side Encryption with Customer provided keys must be made over a secure connection.
	// при использовании собственного ключа MINIO требует TLS клиент-сервер
	sse, err := encrypt.NewSSEC(fileKey)
	if err != nil {
		return
	}

	info, err := minios3.S3PutBytesToFile(ctx, models.MinioClient, bucketName, fname, fileContent, sse)
	if err != nil {
		return
	}
	// зашифровываем ключ файла ключом багета
	objectKey, err := privacy.EncryptB2B(fileKey, bucketKey)
	// переводим в HEX
	objectKeyHex := hex.EncodeToString(objectKey)

	err = db.PutFileParams(ctx, object_id, userName, fname, dataType, objectKeyHex, metaData, int32(info.Size))
	if err != nil {
		return
	}

	return stream.SendAndClose(&pb.ReceiverResponse{
		Success: true,
		Size:    int32(info.Size),
	})
}
