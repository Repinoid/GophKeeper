package minios3

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"gorsovet/internal/models"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/encrypt"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"
)

type TstS3 struct {
	suite.Suite
	t           time.Time
	ctx         context.Context
	minioClient *minio.Client
	// SSE-C (Server-Side Encryption with Customer-Provided Keys)
	sse            encrypt.ServerSide
	minioContainer testcontainers.Container
}

func (suite *TstS3) SetupSuite() { // выполняется перед тестами

	suite.ctx = context.Background()
	suite.t = time.Now()

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic("cannot initialize zap")
	}
	defer logger.Sync()
	models.Sugar = *logger.Sugar() // init global Sugar

	// ***************** MINIO part begin ************************************
	absTLSPath, err := filepath.Abs("../../cmd/tls")
	suite.Require().NoError(err)
	// Запуск контейнера MINIO
	req := testcontainers.ContainerRequest{
		Image:        "minio/minio",
		ExposedPorts: []string{"9000/tcp"},
		Env: map[string]string{
			"MINIO_ROOT_USER":     "minioadmin", // default minioadmin s
			"MINIO_ROOT_PASSWORD": "minioadmin",
			"MINIO_ADDRESS":       ":9000",
		},
		// пробовал порт 9090, не получается, ищет 9000
		Cmd: []string{"server", "--address", ":9000", "/data"},
		HostConfigModifier: func(hostConfig *container.HostConfig) {
			hostConfig.Binds = []string{
				absTLSPath + ":/root/.minio/certs:ro",
				absTLSPath + ":/tls",
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
		ContainerRequest: req,
		Started:          true,
	})
	suite.Require().NoError(err)
	// Terminate в TearDownSuite, дефер не нужен
	//	defer postgresContainer.Terminate(suite.ctx)

	// Получение хоста и порта
	host, err := minioContainer.Host(suite.ctx)
	suite.Require().NoError(err)
	port, err := minioContainer.MappedPort(suite.ctx, "9000")
	suite.Require().NoError(err)
	suite.minioContainer = minioContainer
	models.Sugar.Debugf("Minio доступен по адресу: %s:%s", host, port.Port())

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
	suite.minioClient, err = ConnectToS3(endpoint, "minioadmin", "minioadmin")
	suite.Require().NoError(err)
	// ***************** MINIO part end ************************************

	log.Println("SetupTest() ---------------------")
}

func (suite *TstS3) TearDownSuite() { // // выполняется после всех тестов
	log.Printf("Spent %v\n", time.Since(suite.t))
	suite.minioContainer.Terminate(suite.ctx)
}

func TestS3Suite(t *testing.T) {
	testBase := new(TstS3)
	testBase.ctx = context.Background()

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic("cannot initialize zap")
	}
	defer logger.Sync()
	models.Sugar = *logger.Sugar()

	suite.Run(t, testBase)
}

// ConnectToS3 - get TLS connection to MinIO
func ConnectToTestS3(endpoint string) (client *minio.Client, err error) {

	accessKey := "minioadmin" // auth from docker-compose
	secretKey := "minioadmin"
	useSSL := true // false if no TLS, so endpoint prefix http:// (if true so TLS & https://)

	// // Load CA certificate
	caCert, err := os.ReadFile(models.PublicCrt)
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
		Creds:     credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure:    useSSL,
		Transport: transport,
	})
}
