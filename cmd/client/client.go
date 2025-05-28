package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
		err = PutText(ctx, client, putTextFlag)
		return
	}
	if putFileFlag != "" {
		err = PutFile(ctx, client, putFileFlag)
		return
	}

	if listFlag {
		err = GetListing(ctx, client)
	}

	return
}

func AddUser(ctx context.Context, client pb.GkeeperClient, username, password string) (err error) {
	req := &pb.RegisterRequest{Username: username, Password: password, Metadata: metaFlag}
	resp, err := client.RegisterUser(ctx, req)
	if err != nil {
		return
	}
	models.Sugar.Debugf("%+v", resp)
	return
}

func Login(ctx context.Context, client pb.GkeeperClient, username, password string) (token string, err error) {
	req := &pb.LoginRequest{Username: username, Password: password, Metadata: metaFlag}
	resp, err := client.LoginUser(ctx, req)
	if err != nil {
		return
	}
	token = resp.GetToken()
	if token == "" {
		return "", errors.New("login did not return token")
	}
	// err = os.Setenv("Token", token)
	// if err != nil {
	// 	return
	// }
	models.Sugar.Debugf("%+v", resp.Reply)
	return
}

func PutText(ctx context.Context, client pb.GkeeperClient, text string) (err error) {

	// token, exists := os.LookupEnv("Token")
	// if !exists {
	if token == "" {
		return errors.New("no token")
	}

	reqtxt := &pb.PutTextRequest{Token: token, Textdata: text, Metadata: metaFlag}
	respt, err := client.PutText(ctx, reqtxt)
	models.Sugar.Debugf("%s written %d bytes\n", respt.Reply, respt.Size)

	return
}

func PutFile(ctx context.Context, client pb.GkeeperClient, fpath string) (err error) {
	if token == "" {
		return errors.New("no token")
	}
	fname := filepath.Base(fpath)
	data, err := os.ReadFile(fpath)
	if err != nil {
		return err
	}

	reqtxt := &pb.PutFileRequest{Token: token, Filename: fname, Data: data, Metadata: metaFlag}
	respt, err := client.PutFile(ctx, reqtxt)
	if err != nil {
		models.Sugar.Debugf("client.PutFile  %v\n", err)
		return err
	}
	models.Sugar.Debugf("%s written %d bytes\n", respt.Reply, respt.Size)

	return
}

func GetListing(ctx context.Context, client pb.GkeeperClient) (err error) {
	if token == "" {
		return errors.New("no token")
	}
	reqList := &pb.ListObjectsRequest{Token: token}
	resp, err := client.ListObjects(ctx, reqList)
	if err != nil {
		models.Sugar.Debugf("No listing %v\n", err)
		fmt.Printf("No listing %v\n", err)
		return
	}
	fmt.Printf("%10s\t%10s\t%20s\t%s\n", "ID", "Data type", "created", "metadata")

	list := resp.GetListing()
	for _, v := range list {
		fmt.Printf("%10d\t%10s\t%20s\t%s\n", v.GetId(), v.GetDataType(), (v.GetCreatedAt()).AsTime().Format(time.RFC3339), v.GetMetadata())
	}

	return
}
