package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

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
		str := strings.ReplaceAll(registerFlag, " ", "")
		args := strings.Split(str, ",")
		if len(args) != 2 {
			return errors.New("wrong number of arguments, should be <username, password>")
		}
		re := regexp.MustCompile("^[a-zA-Z0-9]+$")
		if !re.MatchString(args[0]) {
			return errors.New("wrong username, only letters & digits allowed [0-9][a-z][A-Z]")
		}
		err = AddUser(ctx, client, args[0], args[1])
		return
	}

	if loginFlag != "" {
		str := strings.ReplaceAll(loginFlag, " ", "")
		args := strings.Split(str, ",")
		if len(args) != 2 {
			return errors.New("wrong number of arguments, should be <username, password>")
		}
		token := ""
		token, err = Login(ctx, client, args[0], args[1])
		if err := os.WriteFile("token.txt", []byte(token), 0666); err != nil {
			return errors.New("can't write to token.txt")
		}
		return
	}

	if putTextFlag != "" {
		stream, err := client.Greceiver(ctx)
		if err != nil {
			models.Sugar.Debugf("client.Greceiver %v", err)
			return err
		}
		// генерируем случайное имя файла, 8 байт, в HEX распухнет до 16 символов
		forName := make([]byte, 8)
		_, err = rand.Read(forName)
		if err != nil {
			return err
		}
		// переводим в HEX
		objectName := hex.EncodeToString(forName) + ".text"

		// Send text
		resp, err := sendText(stream, putTextFlag, objectName)
		if err != nil || !resp.Success {
			models.Sugar.Debugf("error sending text: %v", err)
			return err
		}
		models.Sugar.Debugf("written %d bytes\n", resp.Size)
		return nil
	}

	if putFileFlag != "" {
		stream, err := client.Greceiver(ctx)
		if err != nil {
			models.Sugar.Debugf("client.Greceiver %v", err)
			return err
		}
		// Send a file
		resp, err := sendFile(stream, putFileFlag)

		if err != nil || !resp.Success {
			models.Sugar.Debugf("error sending file: %v", err)
			return err
		}
		models.Sugar.Debugf("written %d bytes\n", resp.Size)
		return err
	}
	// вывод в терминал списка загруженных юзером объектов
	if listFlag {
		list, err := GetList(ctx, client)
		if err != nil {
			models.Sugar.Debugf("GetList %v", err)
			return err
		}
		fmt.Printf("%10s\t%20s\t%10s\t%15s\t%20s\t%s\n", "ID", "File URL", "Data type", "file size", "created", "metadata")

		for _, v := range list {
			fmt.Printf("%10d\t%20s\t%10s\t%15d\t%20s\t%s\n", v.GetId(), v.GetFileurl(), v.GetDataType(), v.GetSize(),
				(v.GetCreatedAt()).AsTime().Format(time.RFC3339), v.GetMetadata())
		}
	}
	// remove record by it's id
	if removeFlag != 0 {
		fla := removeFlag // для kопипасты
		if !IfIdExist(ctx, client, int32(fla)) {
			fmt.Printf("no record with number %d\n", fla)
			return fmt.Errorf("no record with number %d", fla)
		}
		err = Remover(ctx, client, removeFlag)
		if err != nil {
			models.Sugar.Debugf("Remover %v", err)
			return err
		}
	}
	//
	if showFlag != 0 {
		fla := showFlag
		if !IfIdExist(ctx, client, int32(fla)) {
			fmt.Printf("no record with number %d\n", fla)
			return fmt.Errorf("no record with number %d", fla)
		}

		req := &pb.SenderRequest{ObjectId: int32(showFlag), Token: token}
		stream, err := client.Gsender(ctx, req)
		if err != nil {
			models.Sugar.Debugf("client.Gsender %v", err)
			return err
		}
		by, err := receiveFile(stream)
		if err != nil {
			models.Sugar.Debugf("receiveFile %v", err)
			return err
		}
		fmt.Printf("file %s\nmeta %s\nof type %s\nsize %d\ncreated %s\n",
			by.GetFilename(), by.GetMetadata(), by.GetDataType(), by.GetSize(), by.GetCreatedAt().AsTime().Format(time.RFC3339))
		return nil
	}
	//
	if getFileFlag != 0 {
		fla := getFileFlag
		if !IfIdExist(ctx, client, int32(fla)) {
			fmt.Printf("no record with number %d\n", fla)
			return fmt.Errorf("no record with number %d", fla)
		}
		req := &pb.SenderRequest{ObjectId: int32(getFileFlag), Token: token}
		stream, err := client.Gsender(ctx, req)
		if err != nil {
			models.Sugar.Debugf("client.Gsender %v", err)
			return err
		}
		by, err := receiveFile(stream)
		if err != nil {
			models.Sugar.Debugf("receiveFile %v", err)
			return err
		}
		fileToSave := ""
		if fnameFlag == "" {
			fileToSave = by.GetFilename()
		} else {
			fileToSave = fnameFlag
		}
		if err := os.WriteFile(fileToSave, by.GetContent(), 0666); err != nil {
			return errors.New("can't write to token.txt")
		}
		fmt.Printf("file %s\nmeta %s\nof type %s\nsize %d\ncreated %s\nsaved to %s\n",
			by.GetFilename(), by.GetMetadata(), by.GetDataType(), by.GetSize(), by.GetCreatedAt().AsTime().Format(time.RFC3339), fileToSave)
		return nil
	}

	return
}
