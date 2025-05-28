package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	_ "net/http/pprof"

	pb "gorsovet/cmd/proto"
	"gorsovet/internal/dbase"
	"gorsovet/internal/minio"
	"gorsovet/internal/models"
	"gorsovet/internal/privacy"

	"github.com/minio/minio-go/v7/pkg/encrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (gk *GkeeperService) PutText(ctx context.Context, req *pb.PutTextRequest) (resp *pb.PutTextResponse, err error) {

	// по умолчанию неуспех
	response := pb.PutTextResponse{Success: false, Reply: "Could not put text"}

	// сначала подсоединяемся к Базе Данных, недоступна - отбой
	db, err := dbase.ConnectToDB(ctx, models.DBEndPoint)
	if err != nil {
		models.Sugar.Debugln(err)
		response.Success = false
		response.Reply = "ConnectToDB error"
		return &response, err
	}
	defer db.CloseBase()

	token := req.GetToken()
	userName, err := db.GetUserNameByToken(ctx, token)
	if err != nil {
		return &response, err
	}
	bucketKeyHex, bucketName, err := db.GetBucketKeyByUserName(ctx, userName)
	if err != nil {
		return &response, err
	}
	// в bucketKeyHex - ключ бакета, шифрованный мастер-ключом.  переводим его сначала из HEX в байты
	codedBucketkey, err := hex.DecodeString(bucketKeyHex)
	if err != nil {
		return &response, err
	}
	// deкодируем ключ бакета мастер-ключом
	bucketKey, err := privacy.DecryptB2B(codedBucketkey, models.MasterKey)
	if err != nil {
		return &response, err
	}

	data := req.GetTextdata()
	metadata := req.GetMetadata()

	// создаём случайный ключ для шифрования файла
	fileKey := make([]byte, 32)
	_, err = rand.Read(fileKey)
	if err != nil {
		return &response, err
	}
	// NewSSEC returns a new server-side-encryption using SSE-C and the provided key. The key must be 32 bytes long
	// sse - криптоключ для шифрования файла при записи в Minio
	// Requests specifying Server Side Encryption with Customer provided keys must be made over a secure connection.
	// при использовании собственного ключа требует TLS клиент-сервер
	sse, err := encrypt.NewSSEC(fileKey)

	// генерируем случайное имя файла, 16 байт, в HEX распухнет до 32 символов
	forName := make([]byte, 16)
	_, err = rand.Read(forName)
	if err != nil {
		return &response, err
	}
	// переводим в HEX
	objectName := hex.EncodeToString(forName) + ".text"

	info, err := minio.S3PutBytesToFile(ctx, models.MinioClient, bucketName, objectName, []byte(data), sse)
	if err != nil {
		response.Reply = "bad S3PutBytesToFile"
		return &response, status.Error(codes.Unimplemented, response.Reply)
	}
	// зашифровываем ключ файла ключом багета
	objectKey, err := privacy.EncryptB2B(fileKey, bucketKey)
	// переводим в HEX
	objectKeyHex := hex.EncodeToString(objectKey)

	err = db.PutFileParams(ctx, userName, objectName, "text", objectKeyHex, metadata)
	if err != nil {
		response.Reply = "bad PutFileParams"
		return &response, status.Error(codes.Unimplemented, response.Reply)
	}
	response.Size = info.Size
	response.Success = true
	response.Reply = "OK"

	return &response, err
}
