package dbase

import (
	"fmt"
	"strings"
	"time"

	"context"

	"crypto/rand"
	"encoding/hex"

	pb "gorsovet/cmd/proto"

	"gorsovet/internal/models"
	"gorsovet/internal/privacy"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// AddUser запись нового юзера в таблицу
func (dataBase *DBstruct) AddUser(ctx context.Context, userName, password, metaData string) error {
	db := dataBase.DB

	// транзакция - рудiмент от прошлого проекта, когда данные вносились в пару таблиц. если по итогу ничего не изменится - сверну в простой db.Exec
	tx, err := db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("error db.Begin  %w", err)
	}
	defer tx.Rollback(ctx)

	// генерируем ключ бакета
	bucketKey := make([]byte, 32)
	if _, err = rand.Read(bucketKey); err != nil {
		return err
	}
	// кодируем ключ бакета мастер-ключом
	codedBucketkey, err := privacy.EncryptB2B(bucketKey, models.MasterKey)
	if err != nil {
		return err
	}
	// переводим в HEX
	bucketKeyHex := hex.EncodeToString(codedBucketkey)

	ble := len(bucketKeyHex)
	_ = ble

	// переводим имя в капслок, имя бакета в нижний регистр
	userName = strings.ToUpper(userName)
	bucketname := strings.ToLower(userName)
	order := "INSERT INTO USERA(username, password, bucketname, bucketkey, metadata) VALUES ($1, crypt($2, gen_salt('md5')), $3, $4, $5) ;"
	_, err = tx.Exec(ctx, order, userName, password, bucketname, bucketKeyHex, metaData)
	if err != nil {
		return fmt.Errorf("add user error is %w", err)
	}
	return tx.Commit(ctx)
}

func (dataBase *DBstruct) CheckUserPassword(ctx context.Context, userName, password string) (yes bool) {
	userName = strings.ToUpper(userName)
	db := dataBase.DB
	order := "SELECT (crypt($2, password) = password) FROM USERA WHERE username= $1 ;"
	row := db.QueryRow(ctx, order, userName, password) // password here - what was entered
	//	var yes bool
	// Any error that occurs while querying is deferred until calling Scan on the returned Row.
	// That Row will error with ErrNoRows if no rows are returned.
	err := row.Scan(&yes)
	if err != nil {
		return false
	}
	return
}

func (dataBase *DBstruct) PutToken(ctx context.Context, userName, token, metadata string) (err error) {
	userName = strings.ToUpper(userName)
	db := dataBase.DB

	order := "INSERT INTO TOKENA(userName, token, metadata) VALUES ($1, $2, $3) ;"
	_, err = db.Exec(ctx, order, userName, token, metadata)

	return
}

// IfUserExists возвращает да или нет и id юзера
func (dataBase *DBstruct) IfUserExists(ctx context.Context, userName string) (yes bool, uId int32) {
	userName = strings.ToUpper(userName)
	db := dataBase.DB
	order := "SELECT userId from USERA WHERE username = $1 ;"
	row := db.QueryRow(ctx, order, userName) // password here - what was entered
	//var uId int
	err := row.Scan(&uId)
	return err == nil, uId
}

func (dataBase *DBstruct) GetUserNameByToken(ctx context.Context, token string) (username string, err error) {
	order := "SELECT username from TOKENA WHERE token =  $1 ;"
	row := dataBase.DB.QueryRow(ctx, order, token)
	err = row.Scan(&username)
	return
}

func (dataBase *DBstruct) GetBucketKeyByUserName(ctx context.Context, username string) (bucketKey, bucketName string, err error) {
	order := "SELECT bucketkey, bucketname from USERA WHERE username =  $1 ;"
	row := dataBase.DB.QueryRow(ctx, order, username)
	err = row.Scan(&bucketKey, &bucketName)
	return
}

func (dataBase *DBstruct) PutFileParams(ctx context.Context, username, fileURL, dataType, fileKey, metaData string, fileSize int32) (err error) {

	order := "INSERT INTO DATAS(userName, fileURL, dataType, fileKey, metaData) VALUES ($1, $2, $3, $4, $5, $6) ;"
	_, err = dataBase.DB.Exec(ctx, order, username, fileURL, dataType, fileKey, metaData, fileSize)
	return
}

func (dataBase *DBstruct) GetObjectsList(ctx context.Context, username string) (listing []*pb.ObjectParams, err error) {

	order := "SELECT id, fileURL, datatype, metadata, user_created_at from DATAS WHERE username = $1 order by user_created_at ;"
	rows, err := dataBase.DB.Query(ctx, order, username) //
	if err != nil {
		models.Sugar.Debugf("db.Query %+v\n", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var pgTime time.Time
		ols := pb.ObjectParams{}
		err = rows.Scan(&ols.Id, &ols.Fileurl, &ols.DataType, &ols.Metadata, &pgTime)
		if err != nil {
			models.Sugar.Debugf("rows.Scan %+v\n", err)
			return
		}
		ols.CreatedAt = timestamppb.New(pgTime)
		listing = append(listing, &ols)
	}
	err = rows.Err()
	// Err returns any error that occurred while reading. Err must only be called after the Rows is closed
	if err != nil {
		models.Sugar.Debugf("err = rows.Err() %+v\n", err)
		return
	}

	return
}
