package models

import (
	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"
)

type Carda struct {
	Number     int32  `json:"number"`
	Expiration string `json:"expires"`
	CSV        string `json:"csv"`
	Holder     string `json:"holder"`
}

var (
	MinioClient *minio.Client
	Sugar       zap.SugaredLogger
	Gport       = ":3200"
	DBEndPoint  = "postgres://userp:parole@localhost:5432/dbaza"

	CryptoKey     = []byte("conclave")
	MasterKey     = []byte("Masterkey")
	JWTKey        = []byte("jwtjwtkey")
	PublicCrt     = "../tls/public.crt"
	MinioEndpoint = "localhost:9000"
	MinioUser     = "nail"
	MinioPassword = "password"
)
