package main

import (
	"context"
	"os"

	pb "gorsovet/cmd/proto"
	"gorsovet/internal/models"
	"gorsovet/internal/privacy"

	"google.golang.org/grpc"
)

var gPort = ":3200"
var token = ""

func main() {
	ctx := context.Background()
	err := initClient(ctx)
	if err != nil {
		models.Sugar.Fatal(err)
	}

	if err := run(ctx); err != nil {
		models.Sugar.Info(err)
	}
}

func run(ctx context.Context) (err error) {

	// устанавливаем соединение с сервером
	tlsCreds, err := privacy.LoadClientTLSCredentials("../tls/public.crt")
	if err != nil {
		models.Sugar.Fatalf("cannot load TLS credentials: ", err)
	}
	conn, err := grpc.NewClient(gPort, grpc.WithTransportCredentials(tlsCreds))

	if err != nil {
		models.Sugar.Fatal(err)
	}
	defer conn.Close()

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
