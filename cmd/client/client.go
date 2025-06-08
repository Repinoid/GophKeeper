package main

import (
	"context"
	"fmt"
	"log"
	"os"

	pb "gorsovet/cmd/proto"
	"gorsovet/internal/localbase"
	"gorsovet/internal/models"
	"gorsovet/internal/privacy"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
)

var (
	gPort       = ":3200"
	token       = ""
	localsql    *localbase.LocalDB
	currentUser = "localuser"
)

func main() {
	ctx := context.Background()

	// logger init
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal("cannot initialize zap")
	}
	defer logger.Sync()
	models.Sugar = *logger.Sugar()

	// connect & create tables local DB
	localsql, err = localbase.ConnectToLocalDB(models.LocalSqlEndpoint)
	if err != nil {
		models.Sugar.Errorf("error ConnectToLocalDB  %v", err)
		return
	}
	err = localsql.UsersTableCreation()
	if err != nil {
		models.Sugar.Errorf("error UsersTableCreation  %v", err)
		return
	}
	err = localsql.DataTableCreation()
	if err != nil {
		models.Sugar.Errorf("error DataTableCreation  %v", err)
		return
	}
	// create  S3dir if not exists
	if _, err := os.Stat(models.LocalS3Dir); os.IsNotExist(err) {
		// Create the directory with 0755 permissions (rwx for owner, rx for group/others)
		err := os.Mkdir(models.LocalS3Dir, 0755)
		if err != nil {
			models.Sugar.Errorf("error %s creation %v", models.LocalS3Dir, err)
			return
		}
	}

	// устанавливаем соединение с сервером
	tlsCreds, err := privacy.LoadClientTLSCredentials("../tls/public.crt")
	if err != nil {
		models.Sugar.Fatalf("cannot load TLS credentials: ", err)
	}
	conn, err := grpc.NewClient(gPort, grpc.WithTransportCredentials(tlsCreds))

	// Проверяем состояние соединения - нихрена не работает GetState(), определяем статус сервера через PingServer, это healthcheck
	cr := conn.GetState()
	_ = cr

	isServerErr := PingServer(conn)

	//	if err == nil && conn.GetState() == connectivity.Ready {
	if err == nil && isServerErr == nil {
		// канал открыт, по выходу - закрыть
		defer conn.Close()
		err = initGrpcClient(ctx)
		// если связь с сервером есть но флаги кривые
		if err != nil {
			models.Sugar.Error(err)
			return
		}
		// отработка клиента по GRPC
		if err := runGrpc(ctx, conn); err != nil {
			models.Sugar.Error(err)
			return
		}
		return
	}
	err = initLocalClient(ctx)
	if err != nil {
		models.Sugar.Error(err)
		return
	}
	err = runLocal()
	if err != nil {
		models.Sugar.Error(err)
		return
	}

}

// если сервер доступен
func runGrpc(ctx context.Context, conn *grpc.ClientConn) (err error) {
	// временное решение по хранению токена в файле. создаётся при вызове Login
	tokenB, err := os.ReadFile("token.txt")
	if err == nil {
		token = string(tokenB)
	} else {
		fmt.Println("You are not logged. client -login=\"username, password\"")
		os.Exit(0)
	}

	client := pb.NewGkeeperClient(conn)

	if registerFlag != "" {
		err = registerFlagFunc(ctx, client, registerFlag)
		return
	}

	if loginFlag != "" {
		err = loginFlagFunc(ctx, client, loginFlag)
		return
	}

	if putTextFlag != "" {
		err = putTextFlagFunc(ctx, client, putTextFlag)
		return
	}

	if putFileFlag != "" {
		err = putFileFlagFunc(ctx, client, putFileFlag)
		return err
	}

	// вывод в терминал списка загруженных юзером объектов
	if listFlag {
		err = listFlagFunc(ctx, client)
		return err
	}

	// remove record by it's id
	if removeFlag != 0 {
		err = removeFlagFunc(ctx, client, int32(removeFlag))
		return
	}
	//
	if showFlag != 0 {
		err = showFlagFunc(ctx, client, int32(showFlag))
		return
	}
	//
	if getFileFlag != 0 {
		err = getFileFlagFunc(ctx, client, int32(getFileFlag))
	}

	if putCardFlag != "" {
		// TreatCard засылаем замаршаленные данные карты, в putCardFlag - введённые в CLI c флагом -putcard="...."
		err = sendCard(ctx, client, putCardFlag)
		return err
	}

	return
}

// если сервер в отключке - юзаем локальную базу.
func runLocal() (err error) {
	// имя текущего пользователя
	tokenB, err := os.ReadFile("currentuser.txt")
	if err == nil {
		currentUser = string(tokenB)
	} else {
		fmt.Println("You are not logged. client -login=\"username, password\"")
		os.Exit(0)
	}

	if loginFlag != "" {
		err = loginFlagLocal(loginFlag)
		return
	}

	// вывод в терминал списка загруженных юзером объектов
	if listFlag {
		err = listFlagLocal()
		return err
	}

	//
	if showFlag != 0 {
		err = showFlagLocal(int32(showFlag))
		return
	}
	//
	if getFileFlag != 0 {
		err = getFileFlagLocal(int32(getFileFlag))
	}

	return
}

// PingServer определить самочувствие сервера
func PingServer(conn *grpc.ClientConn) error {
	healthClient := grpc_health_v1.NewHealthClient(conn)
	resp, err := healthClient.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{})
	if err != nil {
		return err
	}

	if resp.Status != grpc_health_v1.HealthCheckResponse_SERVING {
		return fmt.Errorf("server not serving, status: %v", resp.Status)
	}
	return nil
}
