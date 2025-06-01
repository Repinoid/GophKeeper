package minios3

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"math/big"
	"os"

	"gorsovet/internal/models"
	"gorsovet/internal/privacy"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/encrypt"
)

var (
	testBucketName = "testbucket"
	testFileName   = "mi.goo"
)

func (suite *TstS3) Test01Crypt() {

	// Generate your own encryption key (32 bytes)
	key := make([]byte, 32)

	n, err := rand.Read(key)
	suite.Require().NoError(err)
	suite.Require().Equal(n, 32)

	// SSE-C (Server-Side Encryption with Customer-Provided Keys)
	// NewSSEC returns a new server-side-encryption using SSE-C and the provided key. The key must be 32 bytes long
	sse, err := encrypt.NewSSEC(key)
	suite.Require().NoError(err)

	// оказалось что Bucket name cannot be longer than 63 characters
	buck := generateHMAC(key, key)[:53] // левые имена для бакета и файла в нём,  generateHMAC 64 символа
	fil := buck[:50] + "." + buck[50:]

	err = CreateBucket(suite.ctx, suite.minioClient, buck)
	suite.Require().NoError(err)

	// загружаем fil to buck using minio SSE-C encoding
	_, err = suite.minioClient.FPutObject(
		suite.ctx,
		buck,
		fil,
		testFileName,
		minio.PutObjectOptions{
			ServerSideEncryption: sse,
		},
	)
	suite.Require().NoError(err)

	// local file to load in
	gotF := "./got.file"
	// Download SSE-C encrypted object
	err = suite.minioClient.FGetObject(suite.ctx, buck, fil, gotF,
		minio.GetObjectOptions{
			ServerSideEncryption: sse, // same sse object used for upload
		})
	suite.Require().NoError(err)

	// read & compare put & get files
	inFile, err := os.ReadFile(testFileName)
	suite.Require().NoError(err)
	outFile, err := os.ReadFile(gotF)
	suite.Require().NoError(err)
	suite.Require().Equal(inFile, outFile)
	// remove local file
	err = os.Remove(gotF)
	suite.Require().NoError(err)

	// remove  file
	err = S3RemoveFile(suite.ctx, suite.minioClient, buck, fil)
	suite.Require().NoError(err)
	// remove empty bucket
	err = S3RemoveBucket(suite.ctx, suite.minioClient, buck)
	suite.Require().NoError(err)

}

// TestCreateBucket good & ugly
func (suite *TstS3) Test02CreateBucket() {
	err := CreateBucket(suite.ctx, suite.minioClient, testBucketName)
	suite.Require().NoError(err)
	// bucket exists
	err = CreateBucket(suite.ctx, suite.minioClient, testBucketName)
	suite.Require().NoError(err)
	err = CreateBucket(suite.ctx, suite.minioClient, testBucketName+"^&*%$")
	suite.Require().Error(err)
}

// TestWriteRead тест записи бинарных данных в файл и чтения этого файла, крипт свой
func (suite *TstS3) Test03WriteReadBytes() {
	fileContent, err := os.ReadFile(testFileName)
	suite.Require().NoError(err)
	// encrypt "filePath" file
	encrypted, err := privacy.EncryptB2B(fileContent, models.CryptoKey)
	suite.Require().NoError(err)

	_, err = suite.minioClient.PutObject(
		suite.ctx,
		testBucketName,
		"test.file",
		bytes.NewReader(encrypted),
		int64(len(encrypted)),
		minio.PutObjectOptions{},
	)
	suite.Require().NoError(err)
	object, err := suite.minioClient.GetObject(
		suite.ctx,
		testBucketName,
		"test.file",
		minio.GetObjectOptions{},
	)
	suite.Require().NoError(err)
	defer object.Close()
	var buf bytes.Buffer
	_, err = io.Copy(&buf, object)
	suite.Require().NoError(err)

	// Convert to bytes
	fileBytes := buf.Bytes()
	suite.Require().Equal(encrypted, fileBytes)
}

// TestWriteRead тест записи бинарных данных в файл и чтения этого файла
func (suite *TstS3) Test04WriteReadBytes() {

	s3File := "goy.sum"

	data := make([]byte, 2049) // +1b

	n, err := rand.Read(data)
	suite.Require().NoError(err)
	suite.Require().Equal(n, 2049)

	_, err = S3PutBytesToFile(suite.ctx, suite.minioClient, testBucketName, s3File, data, suite.sse)
	suite.Require().NoError(err)

	contentFromS3, err := S3GetFileBytes(suite.ctx, suite.minioClient, testBucketName, s3File, suite.sse)
	suite.Require().NoError(err)

	suite.Require().Equal(contentFromS3, data)

	// remove  file
	err = S3RemoveFile(suite.ctx, suite.minioClient, testBucketName, s3File)
	suite.Require().NoError(err)

}
func (suite *TstS3) Test04WriteReadFile() {

	localFile := "./mi.goo"
	s3File := "go1.sum"

	_, err := S3PutFile(suite.ctx, suite.minioClient, testBucketName, s3File, localFile, suite.sse)
	suite.Require().NoError(err)

	contentFromS3, err := S3GetFileBytes(suite.ctx, suite.minioClient, testBucketName, s3File, suite.sse)
	suite.Require().NoError(err)

	localFileContent, err := os.ReadFile(localFile)
	suite.Require().NoError(err)

	suite.Require().Equal(contentFromS3, localFileContent)

	// remove  file
	err = S3RemoveFile(suite.ctx, suite.minioClient, testBucketName, s3File)
	suite.Require().NoError(err)

}
func (suite *TstS3) Test05Removes() {

	// remove not empty bucket
	err := S3RemoveBucket(suite.ctx, suite.minioClient, testBucketName)
	suite.Require().Error(err)

	// remove  file
	err = S3RemoveFile(suite.ctx, suite.minioClient, testBucketName, "go1.sum")
	suite.Require().NoError(err)

	// remove  file
	err = S3RemoveFile(suite.ctx, suite.minioClient, testBucketName, "test.file")
	suite.Require().NoError(err)

	// remove empty bucket
	err = S3RemoveBucket(suite.ctx, suite.minioClient, testBucketName)
	suite.Require().NoError(err)
}

// TestCrypta проверка собственных функций кодирования и раскодирования
func (suite *TstS3) Test06Crypta() {
	testingString := "\t бывало он ещё в постеле ... © \n"

	coded, err := privacy.EncryptB2B([]byte(testingString), models.CryptoKey)
	suite.Require().NoError(err)

	telo, err := privacy.DecryptB2B(coded, models.CryptoKey)
	suite.Require().NoError(err)

	suite.Require().Equal(testingString, string(telo))
}

func (suite *TstS3) Test07ListObjectsInBucket() {

	buckName := "testbucketforlist"
	err := CreateBucket(suite.ctx, suite.minioClient, buckName)
	suite.Require().NoError(err)

	for i := range 10 {
		fNam := fmt.Sprintf("%02d.tst", i)

		// random file size
		nBig, err := rand.Int(rand.Reader, big.NewInt(11111))
		suite.Require().NoError(err)
		fSize := nBig.Int64()
		// random data
		data := make([]byte, fSize)
		n, err := rand.Read(data)
		suite.Require().NoError(err)
		suite.Require().Equal(int64(n), fSize)
		// write randoms to file
		_, err = S3PutBytesToFile(suite.ctx, suite.minioClient, buckName, fNam, data, suite.sse)
		suite.Require().NoError(err)
	}
	objs, err := S3ListBucket(suite.ctx, suite.minioClient, buckName)
	suite.Require().NoError(err)
	suite.Require().Equal(len(objs), 10)

	for i := range 10 {
		fNam := fmt.Sprintf("%02d.tst", i)
		err = S3RemoveFile(suite.ctx, suite.minioClient, buckName, fNam)
		suite.Require().NoError(err)
	}
	err = S3RemoveBucket(suite.ctx, suite.minioClient, buckName)
	suite.Require().NoError(err)

}

func generateHMAC(key, data []byte) string {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}
