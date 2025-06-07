package main

import (
	"context"
	"log"
	"os"

	pb "gorsovet/cmd/proto"
	"gorsovet/internal/localbase"
	"gorsovet/internal/models"
	"gorsovet/internal/privacy"

	"go.uber.org/zap"
	"google.golang.org/grpc"
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

	if err == nil {
		// канал открыт, по выходу - закрыть
		defer conn.Close()
		err = initGrpcClient(ctx)
		// если связь с сервером есть но флаги кривые
		if err != nil {
			models.Sugar.Fatal(err)
		}
		// отработка клиента по GRPC
		if err := runGrpc(ctx, conn); err != nil {
			models.Sugar.Fatal(err)
		}
		return
	}
	

}

func runGrpc(ctx context.Context, conn *grpc.ClientConn) (err error) {
	// временное решение по хранению токена в файле. создаётся при вызове Login
	tokenB, err := os.ReadFile("token.txt")
	if err == nil {
		token = string(tokenB)
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
