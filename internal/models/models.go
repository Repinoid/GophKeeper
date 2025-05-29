package models

import (
	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"
)

var (
	MinioClient *minio.Client
	Sugar       zap.SugaredLogger
	Gport       = ":3200"
	DBEndPoint  = "postgres://userp:parole@localhost:5432/dbaza"

	CryptoKey = []byte("conclave")
	MasterKey = []byte("Masterkey")
	JWTKey    = []byte("jwtjwtkey")
)
