package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/theplant/luhn"

	pb "gorsovet/cmd/proto"
	"gorsovet/internal/models"
)

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
	models.Sugar.Debugf("%+v", resp.Reply)
	return
}

func GetList(ctx context.Context, client pb.GkeeperClient) (list []*pb.ObjectParams, err error) {
	if token == "" {
		return nil, errors.New("no token")
	}
	reqList := &pb.ListObjectsRequest{Token: token}
	resp, err := client.ListObjects(ctx, reqList)
	if err != nil {
		models.Sugar.Debugf("No listing %v\n", err)
		fmt.Printf("No listing %v\n", err)
		return
	}
	list = resp.GetListing()
	return
}

// IfIdExist проверяем существует ли запись с номером id
func IfIdExist(ctx context.Context, client pb.GkeeperClient, id int32) bool {
	list, err := GetList(ctx, client)
	if err != nil {
		return false
	}
	for _, value := range list {
		if id == value.GetId() {
			return true
		}
	}
	return false
}

// Remover удаляем запись (ака объект) с номером id
func Remover(ctx context.Context, client pb.GkeeperClient, id int) (err error) {
	if token == "" {
		return errors.New("no token")
	}

	req := &pb.RemoveObjectsRequest{ObjectId: int32(id), Token: token}
	resp, err := client.RemoveObjects(ctx, req)
	if err != nil {
		models.Sugar.Debugf("No listing %v\n", err)
		fmt.Printf("No listing %v\n", err)
		return
	}
	if !resp.Success {
		return fmt.Errorf("could not delete object number %d", id)
	}

	return
}

// receiveFile - получаем запись из хранилища
func receiveFile(stream pb.Gkeeper_GsenderClient) (chuvak *pb.SenderChunk, err error) {
	if token == "" {
		return nil, errors.New("no token")
	}
	// в первом куске помимо данных - параметры записи
	chu := pb.SenderChunk{}
	firstChunk, err := stream.Recv()
	if err != nil {
		models.Sugar.Debugf("stream.Recv()  %v", err)
		return nil, err
	}
	chu.Content = firstChunk.GetContent()
	chu.Filename = firstChunk.GetFilename()
	chu.Metadata = firstChunk.GetMetadata()
	chu.Size = firstChunk.GetSize()
	chu.CreatedAt = firstChunk.GetCreatedAt()
	chu.DataType = firstChunk.GetDataType()

	// Process subsequent chunks
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		chu.Content = append(chu.Content, chunk.GetContent()...)
	}
	return &chu, err
}

func TreatCard(ctx context.Context, client pb.GkeeperClient, cardData string) (err error) {
	args := strings.Split(cardData, ",")
	if len(args) != 4 {
		return errors.New("wrong number of arguments, should be cardnumber digits,expiration MM/YY, CSV, cardholder name\"")
	}
	cardNum := strings.ReplaceAll(args[0], " ", "")
	cnumi, err := strconv.Atoi(cardNum)

	if !luhn.Valid(cnumi) || err != nil {
		return errors.New("wrong Card Number. Not real")
	}
	exp := strings.ReplaceAll(args[1], " ", "")
	// should use raw string (`...`) with regexp.MustCompile to avoid having to escape twice (S1007)
	re := regexp.MustCompile(`^\d\d/\d\d$`) // MM/YY
	if !re.MatchString(exp) {
		return errors.New("wrong Card Number. Not real")
	}
	csv := strings.ReplaceAll(args[2], " ", "")
	re = regexp.MustCompile(`^\d\d\d$`) // CSV 3 digits
	if !re.MatchString(csv) {
		return errors.New("wrong CSV. Proposed to be 3 digits")
	}
	holder := strings.TrimSpace(args[3])
	holder = strings.ReplaceAll(holder, "  ", " ")
	re = regexp.MustCompile(`^[a-zA-Z\s]+$`)
	if !re.MatchString(holder) {
		fmt.Printf("this mazafaka does not exist. Only latin symbols are allowed %s\n", holder)
	}
	card := models.Carda{Number: int32(cnumi), Expiration: exp, CSV: csv, Holder: holder}
	marsh, err := json.Marshal(card)
	if err != nil {
		return err
	}
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
	objectName := hex.EncodeToString(forName) + ".card"

	// Send text
	resp, err := sendText(stream, string(marsh), objectName, "card")
	if err != nil || !resp.Success {
		models.Sugar.Debugf("error sending card data: %v", err)
		return err
	}
	models.Sugar.Debugf("written %d bytes\n", resp.Size)
	return nil

}
