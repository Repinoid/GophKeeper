package minio

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"gorsovet/internal/models"
	"io"
	"net/http"
	"os"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/encrypt"
)

// ConnectToS3 - get TLS connection to MinIO
func ConnectToS3() (client *minio.Client, err error) {

	endpoint := "localhost:9000"
	accessKey := "nail"
	secretKey := "password"
	useSSL := true // true if TLS, so endpoint prefix https://

	// Load CA certificate
	caCert, err := os.ReadFile("../../internal/certs/public.crt")
	if err != nil {
		return nil, fmt.Errorf("error reading CA certificate: %w", err)
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	// Configure TLS
	tlsConfig := &tls.Config{
		RootCAs:            caCertPool,
		InsecureSkipVerify: false, // Set to true only for testing with self-signed certs
	}

	// Initialize minio client object with custom transport
	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}
	return minio.New(endpoint, &minio.Options{
		//client, err = minio.New(endpoint, &minio.Options{
		Creds:     credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure:    useSSL,
		Transport: transport,
	})
	// transport.CloseIdleConnections()

	//return
}

// CreateBucket - create new bucket if not exist
func CreateBucket(ctx context.Context, minioClient *minio.Client, bucketName string) (err error) {

	exists, err := minioClient.BucketExists(ctx, bucketName)
	if exists {
		models.Sugar.Debugf("buckect %s exists\n", bucketName)
		return nil
	}
	// if ошибка вызова BucketExists
	if err != nil {
		models.Sugar.Debugf("Bucket %s BucketExists method error: %v", bucketName, err)
		return fmt.Errorf("bucket %s BucketExists error: %w", bucketName, err)
	}
	err = minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
	return
}

// S3PutBytesToFile write []bute to "bucketName/objectName"
func S3PutBytesToFile(ctx context.Context, minioClient *minio.Client,
	bucketName, objectName string, data []byte, sse encrypt.ServerSide) (info minio.UploadInfo, err error) {

	info, err = minioClient.PutObject(
		ctx,
		bucketName,
		objectName,
		bytes.NewReader(data),
		int64(len(data)),
		minio.PutObjectOptions{ServerSideEncryption: sse},
	)
	if err != nil {
		return
	}
	models.Sugar.Debugf("file written lenght %d\n", info.Size)
	// Check if file exists on minio
	_, err = minioClient.StatObject(ctx, bucketName, objectName, minio.StatObjectOptions{ServerSideEncryption: sse})
	return
}

// S3PutFile   Upload "filePath" file to "bucketName/objectName"
func S3PutFile(ctx context.Context, minioClient *minio.Client,
	bucketName, objectName, filePath string, sse encrypt.ServerSide) (info minio.UploadInfo, err error) {

	info, err = minioClient.FPutObject(
		ctx,
		bucketName,
		objectName,
		filePath,
		minio.PutObjectOptions{ServerSideEncryption: sse},
	)
	if err != nil {
		return
	}
	models.Sugar.Debugf("file written lenght %d\n", info.Size)
	// Check if file exists on minio
	_, err = minioClient.StatObject(ctx, bucketName, objectName, minio.StatObjectOptions{ServerSideEncryption: sse})
	return
}

func S3GetFileBytes(ctx context.Context, minioClient *minio.Client,
	bucketName, objectName string, sse encrypt.ServerSide) (fileBytes []byte, err error) {

	// Get the object from MinIO
	object, err := minioClient.GetObject(
		context.Background(),
		bucketName,
		objectName,
		minio.GetObjectOptions{ServerSideEncryption: sse},
	)
	if err != nil {
		return
	}
	defer object.Close()
	// Read the object's data into a byte slice
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, object); err != nil {
		return nil, fmt.Errorf("failed to read object data: %w", err)
	}
	// Convert to bytes
	fileBytes = buf.Bytes()

	models.Sugar.Debugf("file read lenght %d\n", len(fileBytes))

	return fileBytes, err
}

func S3RemoveFile(ctx context.Context, minioClient *minio.Client, bucketName, objectName string) (err error) {
	err = minioClient.RemoveObject(ctx, bucketName, objectName, minio.RemoveObjectOptions{})
	models.Sugar.Debugf("S3RemoveFile %s from %s err %v\n", objectName, bucketName, err)
	return
}
func S3RemoveBucket(ctx context.Context, minioClient *minio.Client, bucketName string) (err error) {
	err = minioClient.RemoveBucket(ctx, bucketName)
	models.Sugar.Debugf("S3RemoveBucket %s  err %v\n", bucketName, err)
	return
}

func S3ListBucket(ctx context.Context, minioClient *minio.Client, bucketName string) (objs []minio.ObjectInfo, err error) {
	// var objectCh <-chan minio.ObjectInfo
	objectCh := minioClient.ListObjects(ctx, bucketName, minio.ListObjectsOptions{
		Recursive: false,
	})
	for object := range objectCh {
		if object.Err != nil {
			return nil, object.Err
		}
		objs = append(objs, object)
	}
	return
}
