package main

import (
	"context"
	"crypto/rand"
	"log"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/encrypt"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

type TstS3 struct {
	suite.Suite
	t           time.Time
	ctx         context.Context
	minioClient *minio.Client
	// SSE-C (Server-Side Encryption with Customer-Provided Keys)
	sse encrypt.ServerSide
}

func (suite *TstS3) SetupSuite() { // выполняется перед тестами
	suite.ctx = context.Background()
	suite.t = time.Now()

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic("cannot initialize zap")
	}
	defer logger.Sync()
	Sugar = *logger.Sugar() // init global Sugar

	// Generate your own encryption key (32 bytes)
	key := make([]byte, 32)
	n, err := rand.Read(key)
	suite.Require().NoError(err)
	suite.Require().Equal(n, 32)

	//
	// NewSSEC returns a new server-side-encryption using SSE-C and the provided key. The key must be 32 bytes long
	suite.sse, err = encrypt.NewSSEC(key)
	suite.Require().NoError(err)

	// Best Practices - Reuse the client: Create one client instance and reuse it throughout your application.
	suite.minioClient, err = ConnectToS3()
	suite.Require().NoError(err)

	log.Println("SetupTest() ---------------------")
}

func (suite *TstS3) TearDownSuite() { // // выполняется после всех тестов
	log.Printf("Spent %v\n", time.Since(suite.t))
}

func TestS3Suite(t *testing.T) {
	testBase := new(TstS3)
	testBase.ctx = context.Background()

	suite.Run(t, testBase)

}
