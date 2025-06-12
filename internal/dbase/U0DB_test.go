package dbase

import (
	"context"
	"crypto/rand"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"gorsovet/internal/minios3"
	"gorsovet/internal/models"

	"go.uber.org/zap"

	"github.com/docker/docker/api/types/container"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/encrypt"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type TstBase struct {
	suite.Suite
	t   time.Time
	ctx context.Context
	//	dataBase          *DBstruct
	DBEndPoint        string
	postgresContainer testcontainers.Container

	minioClient *minio.Client
	// SSE-C (Server-Side Encryption with Customer-Provided Keys)
	sse            encrypt.ServerSide
	minioContainer testcontainers.Container
}

func (suite *TstBase) SetupSuite() { // выполняется перед тестами
	suite.ctx = context.Background()
	suite.t = time.Now()

	// ***************** POSTGREs part begin ************************************
	// Запуск контейнера PostgreSQL
	req := testcontainers.ContainerRequest{
		Image:        "postgres:17",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_PASSWORD": "testpass",
			"POSTGRES_USER":     "testuser",
			"POSTGRES_DB":       "testdb",
		},
		WaitingFor: wait.ForListeningPort("5432/tcp"),
	}

	postgresContainer, err := testcontainers.GenericContainer(suite.ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	suite.Require().NoError(err)
	//	defer postgresContainer.Terminate(suite.ctx)

	// Получение хоста и порта
	host, err := postgresContainer.Host(suite.ctx)
	suite.Require().NoError(err)
	port, err := postgresContainer.MappedPort(suite.ctx, "5432")
	suite.Require().NoError(err)
	suite.DBEndPoint = fmt.Sprintf("postgres://testuser:testpass@%s:%s/testdb", host, port.Port())
	suite.postgresContainer = postgresContainer
	models.Sugar.Debugf("PostgreSQL доступен по адресу: %s:%s", host, port.Port())
	// ***************** POSTGREs part end ************************************

	// ***************** MINIO part begin ************************************
	absTLSPath, err := filepath.Abs("../../cmd/tls")
	suite.Require().NoError(err)
	// Запуск контейнера MINIO
	reqm := testcontainers.ContainerRequest{
		Image:        "minio/minio",
		ExposedPorts: []string{"9000/tcp"},
		Env: map[string]string{
			"MINIO_ROOT_USER":     "minioadmin", // default minioadmin s
			"MINIO_ROOT_PASSWORD": "minioadmin",
			//	"MINIO_ADDRESS":       ":9000",
		},
		// пробовал порт 9090, не получается, ищет 9000
		Cmd: []string{"server", "--address", ":9000", "/data"},
		HostConfigModifier: func(hostConfig *container.HostConfig) {
			hostConfig.Binds = []string{
				absTLSPath + ":/root/.minio/certs:ro",
			}
			// hostConfig.PortBindings = nat.PortMap{
			// 	"9000/tcp": []nat.PortBinding{
			// 		{
			// 			HostIP:   "0.0.0.0",
			// 			HostPort: "9000", // Bind to same port on host
			// 		},
			// 	},
			// }
		},
		WaitingFor: wait.ForAll(
			wait.ForLog("API:"),
			wait.ForListeningPort("9000/tcp"),
		),
	}
	minioContainer, err := testcontainers.GenericContainer(suite.ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: reqm,
		Started:          true,
	})
	suite.Require().NoError(err)
	// Terminate в TearDownSuite, дефер не нужен
	//	defer postgresContainer.Terminate(suite.ctx)

	// Получение хоста и порта
	hostm, err := minioContainer.Host(suite.ctx)
	suite.Require().NoError(err)
	portm, err := minioContainer.MappedPort(suite.ctx, "9000")
	suite.Require().NoError(err)
	suite.minioContainer = minioContainer
	models.Sugar.Debugf("Minio доступен по адресу: %s:%s", hostm, portm.Port())

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
	//endpoint := fmt.Sprintf("%s:%s", host, port.Port())
	endpoint, err := minioContainer.Endpoint(suite.ctx, "")
	suite.Require().NoError(err)
	// переназначаем путь к ключу, по умолсению в models.PublicCrt запуск из cmd/client
	models.PublicCrt = "../../cmd/tls/public.crt"
	suite.minioClient, err = minios3.ConnectToS3(endpoint, "minioadmin", "minioadmin")
	suite.Require().NoError(err)
	// клиент для функций минио в models.MinioClient
	models.MinioClient = suite.minioClient
	// ***************** MINIO part end ************************************

	models.Sugar.Infoln("SetupTest() ---------------------")
}

func (suite *TstBase) TearDownSuite() { // // выполняется после всех тестов
	models.Sugar.Infof("Spent %v\n", time.Since(suite.t))
	//	suite.dataBase.CloseBase()
	// прикрываем контейнер с БД
	suite.postgresContainer.Terminate(suite.ctx)
}

func TestHandlersSuite(t *testing.T) {
	testBase := new(TstBase)
	testBase.ctx = context.Background()

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic("cannot initialize zap")
	}
	defer logger.Sync()
	models.Sugar = *logger.Sugar()

	models.Sugar.Infoln("before run ....")
	suite.Run(t, testBase)

}
