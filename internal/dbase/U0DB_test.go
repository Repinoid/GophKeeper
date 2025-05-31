package dbase

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"
)

var Sugar zap.SugaredLogger

type TstBase struct {
	suite.Suite
	t   time.Time
	ctx context.Context
	//	dataBase          *DBstruct
	DBEndPoint        string
	postgresContainer testcontainers.Container
}

func (suite *TstBase) SetupSuite() { // выполняется перед тестами
	suite.ctx = context.Background()
	suite.t = time.Now()

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
	Sugar.Debugf("PostgreSQL доступен по адресу: %s:%s", host, port.Port())

	Sugar.Infoln("SetupTest() ---------------------")
}

func (suite *TstBase) TearDownSuite() { // // выполняется после всех тестов
	Sugar.Infof("Spent %v\n", time.Since(suite.t))
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
	Sugar = *logger.Sugar()

	Sugar.Infoln("before run ....")
	suite.Run(t, testBase)

}
