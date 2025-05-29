package models

import (
	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"
)

var (
<<<<<<< HEAD
	MinioClient *minio.Client
	Sugar       zap.SugaredLogger
	Gport       = ":3200"
	DBEndPoint  = "postgres://userp:parole@localhost:5432/dbaza"

	CryptoKey = []byte("conclave")
	MasterKey = []byte("Masterkey")
	JWTKey    = []byte("jwtjwtkey")
=======
	Sugar zap.SugaredLogger
	Gport = ":3200"
	// password minimum 8 symbols !!!!
	DBEndPoint = "postgres://userw:myparole@localhost:5432/baza"
>>>>>>> origin/main
)
