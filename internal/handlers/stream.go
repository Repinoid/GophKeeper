package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	_ "net/http/pprof"

	pb "gorsovet/cmd/proto"
	"gorsovet/internal/dbase"
	"gorsovet/internal/minio"
	"gorsovet/internal/models"
	"gorsovet/internal/privacy"

	"github.com/minio/minio-go/v7/pkg/encrypt"
)

// Gkeeper_ProbaFuncServer
// func (gk *GkeeperService) ProbaFunc(req *pb.Chunk, stream pb.Gkeeper_ProbaFuncServer) (err error) {
func (gk *GkeeperService) ProbaFunc(stream pb.Gkeeper_ProbaFuncServer) (err error) {
	ctx := context.Background()
	fname := req.GetFilename()
	token := req.GetToken()

	db, err := dbase.ConnectToDB(ctx, models.DBEndPoint)
	if err != nil {
		models.Sugar.Debugf("ConnectToDB  %v", err)
		return
	}
	defer db.CloseBase()

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
	metadata := req.GetMetadata()
	// создаём случайный ключ для шифрования файла
	fileKey := make([]byte, 32)
	_, err = rand.Read(fileKey)
	if err != nil {
		return
	}
	// NewSSEC returns a new server-side-encryption using SSE-C and the provided key. The key must be 32 bytes long
	// sse - криптоключ для шифрования файла при записи в Minio
	// Requests specifying Server Side Encryption with Customer provided keys must be made over a secure connection.
	// при использовании собственного ключа требует TLS клиент-сервер
	sse, err := encrypt.NewSSEC(fileKey)

	fileContent := []byte{}
	// Process subsequent chunks
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
	// info !!!
	_, err = minio.S3PutBytesToFile(ctx, models.MinioClient, bucketName, fname, fileContent, sse)
	if err != nil {
		return
	}
	// зашифровываем ключ файла ключом багета
	objectKey, err := privacy.EncryptB2B(fileKey, bucketKey)
	// переводим в HEX
	objectKeyHex := hex.EncodeToString(objectKey)

	err = db.PutFileParams(ctx, userName, fname, "file", objectKeyHex, metadata)
	if err != nil {
		return
	}

	return
}

func (gk *GkeeperService) UploadFile(stream pb.Gkeeper_UploadFileServer) (err error) {

	ctx := context.Background()
	// First message should contain the filename
	firstChunk, err := stream.Recv()
	if err != nil {
		models.Sugar.Debugf("stream.Recv()  %v", err)
		return err
	}
	// get file name from first chunk
	fname := firstChunk.GetFilename()
	fileContent := firstChunk.GetContent()

	// затем подсоединяемся к Базе Данных, недоступна - отбой
	db, err := dbase.ConnectToDB(ctx, models.DBEndPoint)
	if err != nil {
		models.Sugar.Debugf("ConnectToDB  %v", err)
		return
	}
	defer db.CloseBase()

	token := firstChunk.GetToken()
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
	metadata := firstChunk.GetMetadata()

	// создаём случайный ключ для шифрования файла
	fileKey := make([]byte, 32)
	_, err = rand.Read(fileKey)
	if err != nil {
		return
	}
	// NewSSEC returns a new server-side-encryption using SSE-C and the provided key. The key must be 32 bytes long
	// sse - криптоключ для шифрования файла при записи в Minio
	// Requests specifying Server Side Encryption with Customer provided keys must be made over a secure connection.
	// при использовании собственного ключа требует TLS клиент-сервер
	sse, err := encrypt.NewSSEC(fileKey)

	// Process subsequent chunks
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		fileContent = append(fileContent, chunk.GetContent()...)

		// Optional: Send acknowledgment back to client
		if err := stream.Send(&pb.Chunk{
			Content: []byte("chunk received ..."),
		}); err != nil {
			return err
		}
	}
	info, err := minio.S3PutBytesToFile(ctx, models.MinioClient, bucketName, fname, fileContent, sse)
	if err != nil {
		return
	}
	// зашифровываем ключ файла ключом багета
	objectKey, err := privacy.EncryptB2B(fileKey, bucketKey)
	// переводим в HEX
	objectKeyHex := hex.EncodeToString(objectKey)

	err = db.PutFileParams(ctx, userName, fname, "file", objectKeyHex, metadata)
	if err != nil {
		return
	}

	sent := fmt.Sprintf("file %s size %d\n", fname, info.Size)
	err = stream.Send(&pb.Chunk{Content: []byte(sent)})

	return
}
