package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"

	pb "gorsovet/cmd/proto"
	"gorsovet/internal/models"
)

func sendFile(stream pb.Gkeeper_GreceiverClient, fpath, objectName string) (resp *pb.ReceiverResponse, err error) {

	if token == "" {
		return nil, errors.New("no token")
	}

	file, err := os.Open(fpath)
	if err != nil {
		return
	}
	defer file.Close()

	buffer := make([]byte, 64*1024) // 64KB chunks

	// Send first chunk with filename etc
	firstChunk := &pb.ReceiverChunk{Filename: objectName, Token: token, Metadata: metaFlag, DataType: "file", ObjectId: int32(updateFlag)}
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return
	}
	firstChunk.Content = buffer[:n]

	if err = stream.Send(firstChunk); err != nil {
		return
	}
	// Send remaining chunks
	for {
		n, err := file.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if err := stream.Send(&pb.ReceiverChunk{
			Content: buffer[:n],
		}); err != nil {
			return nil, err
		}
	}
	resp, err = stream.CloseAndRecv()
	return
}

func sendText(stream pb.Gkeeper_GreceiverClient, text, objectName string, dtype string) (resp *pb.ReceiverResponse, err error) {

	if token == "" {
		return nil, errors.New("no token")
	}

	reader := strings.NewReader(text)

	buffer := make([]byte, 64*1024) // 64KB chunks

	// Send first chunk with filename
	firstChunk := &pb.ReceiverChunk{Filename: objectName, Token: token, Metadata: metaFlag, DataType: dtype, ObjectId: int32(updateFlag)}
	n, err := reader.Read(buffer)
	if err != nil && err != io.EOF {
		return
	}
	firstChunk.Content = buffer[:n]

	if err = stream.Send(firstChunk); err != nil {
		return
	}
	// Send remaining chunks
	for {
		n, err := reader.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if err := stream.Send(&pb.ReceiverChunk{
			Content: buffer[:n],
		}); err != nil {
			return nil, err
		}
	}
	resp, err = stream.CloseAndRecv()
	return
}

func sendCard(ctx context.Context, client pb.GkeeperClient, cardData string) (err error) {
	args := strings.Split(cardData, ",")
	if len(args) != 4 {
		return errors.New("wrong number of arguments, should be cardnumber digits, expiration MM/YY, CSV, cardholder name\"")
	}
	cardNumStr := strings.ReplaceAll(args[0], " ", "")
	cnumi, err := strconv.ParseInt(cardNumStr, 10, 64)
	// номер карты должен быть не менее 13 и не более 19 цифр, удовлетворять алгоритму Луна
	if len(cardNumStr) < 13 || len(cardNumStr) > 19 || !LuhnCheck(cardNumStr) || err != nil {
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
		return fmt.Errorf("this mazafaka does not exist. Only latin symbols are allowed %s", holder)
	}
	// маршаллим данные карты
	card := models.Carda{Number: cnumi, Expiration: exp, CSV: csv, Holder: holder}
	marsh, err := json.Marshal(card)
	if err != nil {
		return err
	}
	stream, err := client.Greceiver(ctx)
	if err != nil {
		models.Sugar.Debugf("client.Greceiver %v", err)
		return err
	}
	// генерируем случайный префикс имени файла, 4 байт, в HEX распухнет до 8 символов
	forName := make([]byte, 4)
	_, err = rand.Read(forName)
	if err != nil {
		return err
	}
	// переводим в HEX, add ****, add last 4 card digits
	objectName := hex.EncodeToString(forName) + "****" + cardNumStr[len(cardNumStr)-4:] + ".card"

	// Send card params as marshalled text with DataType "card"
	resp, err := sendText(stream, string(marsh), objectName, "card")
	if err != nil || !resp.Success {
		models.Sugar.Debugf("error sending card data: %v", err)
		return err
	}
	models.Sugar.Debugf("written %d bytes\n", resp.Size)
	return nil

}

// карта перекрёстка (старая если чё)
// 5303 3131 5442 5748

// LuhnCheck проверяет номер карты по алгоритму Луна
func LuhnCheck(cardNumber string) bool {
	// Удаляем все пробелы и нецифровые символы
	cardNumber = strings.ReplaceAll(cardNumber, " ", "")

	sum := 0
	alternate := false

	// Проходим по цифрам справа налево
	for i := len(cardNumber) - 1; i >= 0; i-- {
		digit, err := strconv.Atoi(string(cardNumber[i]))
		if err != nil {
			return false // Нечисловой символ
		}

		if alternate {
			digit *= 2
			if digit > 9 {
				digit = (digit % 10) + 1
			}
		}

		sum += digit
		alternate = !alternate
	}

	// Номер карты валиден, если сумма кратна 10
	return sum%10 == 0
}
