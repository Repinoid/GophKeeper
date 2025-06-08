package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	pb "gorsovet/cmd/proto"
	"gorsovet/internal/localbase"
	"gorsovet/internal/models"
)

func registerFlagFunc(ctx context.Context, client pb.GkeeperClient, registerFlag string) (err error) {

	str := strings.ReplaceAll(registerFlag, " ", "")
	args := strings.Split(str, ",")
	if len(args) != 2 {
		return errors.New("wrong number of arguments, should be <username, password>")
	}
	re := regexp.MustCompile("^[a-zA-Z0-9]+$")
	if !re.MatchString(args[0]) {
		return errors.New("wrong username, only letters & digits allowed [0-9][a-z][A-Z]")
	}
	if len(args[0]) < 3 {
		return errors.New("username cannot be shorter than 3 characters")
	}
	err = AddUser(ctx, client, args[0], args[1])
	if err != nil {
		return err
	}
	err = localbase.AddUser(*localsql, args[0], args[1])
	return
}

func loginFlagFunc(ctx context.Context, client pb.GkeeperClient, loginFlag string) (err error) {
	str := strings.ReplaceAll(loginFlag, " ", "")
	args := strings.Split(str, ",")
	if len(args) != 2 {
		return errors.New("wrong number of arguments, should be <username, password>")
	}
	token := ""
	token, err = Login(ctx, client, args[0], args[1])
	if err != nil {
		fmt.Println("Wrong username/password")
		os.Exit(0)
	}
	// сохраняем токен локально
	if err := os.WriteFile("token.txt", []byte(token), 0666); err != nil {
		return errors.New("can't write to token.txt")
	}
	currentUser = args[0]
	return
}

func putTextFlagFunc(ctx context.Context, client pb.GkeeperClient, putTextFlag string) (err error) {
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
	resp, err := sendText(stream, putTextFlag, objectName, "text")
	if err != nil || !resp.Success {
		models.Sugar.Debugf("error sending text: %v", err)
		return err
	}
	models.Sugar.Debugf("written %d bytes\n", resp.Size)
	return nil
}

func putFileFlagFunc(ctx context.Context, client pb.GkeeperClient, putFileFlag string) (err error) {
	stream, err := client.Greceiver(ctx)
	if err != nil {
		models.Sugar.Debugf("client.Greceiver %v", err)
		return err
	}
	// получаем имя файла без пути
	fname := filepath.Base(putFileFlag)
	// генерируем случайный префикс для имени файла, 4 байта, в HEX распухнет до 8 символов
	forName := make([]byte, 8)
	_, err = rand.Read(forName)
	if err != nil {
		return err
	}
	// переводим в HEX
	objectName := hex.EncodeToString(forName) + "_" + fname

	// Send a file
	resp, err := sendFile(stream, putFileFlag, objectName)

	if err != nil || !resp.Success {
		models.Sugar.Debugf("error sending file: %v", err)
		return err
	}
	models.Sugar.Debugf("written %d bytes\n", resp.Size)
	return
}

func listFlagFunc(ctx context.Context, client pb.GkeeperClient) (err error) {
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
	return
}

func removeFlagFunc(ctx context.Context, client pb.GkeeperClient, removeFlag int32) (err error) {

	fla := removeFlag
	if !IfIdExist(ctx, client, fla) {
		fmt.Printf("no record with number %d\n", fla)
		return fmt.Errorf("no record with number %d", fla)
	}
	err = Remover(ctx, client, removeFlag)
	if err != nil {
		models.Sugar.Debugf("Remover %v", err)
		return err
	}
	err = localbase.Remover(*localsql, removeFlag)
	if err != nil {
		models.Sugar.Debugf("Local Remover %v", err)
		return err
	}

	return
}

func showFlagFunc(ctx context.Context, client pb.GkeeperClient, showFlag int32) (err error) {
	fla := showFlag
	if !IfIdExist(ctx, client, fla) {
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
	fmt.Printf("file:\t%s\nmeta:\t%s\ntype:\t%s\nsize:\t%d\ncreated:\t%s\n",
		by.GetFilename(), by.GetMetadata(), by.GetDataType(), by.GetSize(), by.GetCreatedAt().AsTime().Format(time.RFC3339))

	if by.GetDataType() == "text" {
		fmt.Println("_____________________________________________________________________________ CONTENT __")
		fmt.Println(string(by.GetContent()))
	}
	if by.GetDataType() == "card" {
		fmt.Println("_____________________________________________________________________________ CARD __")
		cardRaw := by.GetContent()

		card := models.Carda{}
		err := json.Unmarshal(cardRaw, &card)
		if err != nil {
			return err
		}
		fmt.Printf("Number     %20d\nExpiration %20s\nCSV        %20s\nHolder     %20s\n", card.Number, card.Expiration, card.CSV, card.Holder)
	}
	return nil
}

func getFileFlagFunc(ctx context.Context, client pb.GkeeperClient, getFileFlag int32) (err error) {

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
	fmt.Printf("file:\t%s\nmeta:\t%s\ntype:\t%s\nsize:\t%d\ncreated:\t%s\nsaved to:\t%s\n",
		by.GetFilename(), by.GetMetadata(), by.GetDataType(), by.GetSize(), by.GetCreatedAt().AsTime().Format(time.RFC3339), fileToSave)
	return nil
}
