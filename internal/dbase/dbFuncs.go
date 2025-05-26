package dbase

import (
	"fmt"
	"strings"

	"context"

	"crypto/rand"
	"encoding/hex"

	"gorsovet/internal/models"
	"gorsovet/internal/privacy"
)

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

// nil - user exists
func (dataBase *DBstruct) IfUserExists(ctx context.Context, userName string) (yes bool, uId int32) {
	userName = strings.ToUpper(userName)
	db := dataBase.DB
	order := "SELECT userId from USERA WHERE username = $1 ;"
	row := db.QueryRow(ctx, order, userName) // password here - what was entered
	//var uId int
	err := row.Scan(&uId)
	return err == nil, uId
}
