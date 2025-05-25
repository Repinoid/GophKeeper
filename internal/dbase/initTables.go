package dbase

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"gorsovet/internal/models"
	"gorsovet/internal/privacy"
)

type DBstruct struct {
	//	DB     *pgx.Conn
	DB *pgxpool.Pool
	// UserID int64
}

var (
	DBEndPoint string
	MasterKey  []byte = []byte("MasterKey")
	Sugar      zap.SugaredLogger
)

// UsersTableCreation создание таблицы юзеров
func (dataBase *DBstruct) UsersTableCreation(ctx context.Context) error {

	_ = models.Gport

	db := dataBase.DB
	_, err := db.Exec(ctx, "CREATE EXTENSION IF NOT EXISTS pgcrypto;") // расширение для хэширования паролей
	if err != nil {
		return fmt.Errorf("CREATE EXTENSION pgcrypto; %w", err)
	}

	creatorOrder :=
		"CREATE TABLE IF NOT EXISTS USERA" +
			"(userId INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY, " +
			"username VARCHAR(64) UNIQUE, " +
			"password TEXT NOT NULL, " +
			"bucketname VARCHAR(64) NOT NULL, " +
			"bucketkey TEXT NOT NULL, " +
			"metadata TEXT, " +
			"roles int, " +
			"user_created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP);"

	_, err = db.Exec(ctx, creatorOrder)
	if err != nil {
		return fmt.Errorf("create users table. %w", err)
	}
	return nil
}

// ConnectToDB получить эндпоинт Базы Данных
func ConnectToDB(ctx context.Context, DBEndPoint string) (dataBase *DBstruct, err error) {

	//	baza, err := pgx.Connect(ctx, DBEndPoint)
	baza, err := pgxpool.New(ctx, DBEndPoint)
	if err != nil {
		return nil, fmt.Errorf("can't connect to DB %s err %w", DBEndPoint, err)
	}
	dataBase = &DBstruct{DB: baza} // Initialize

	if err := dataBase.UsersTableCreation(ctx); err != nil {
		return nil, err
	}
	return
}

func (dataBase *DBstruct) CloseBase() {
	dataBase.DB.Close()
}

func (dataBase *DBstruct) AddUser(ctx context.Context, userName, password, metaData string) error {
	db := dataBase.DB

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
	codedBucketkey, err := privacy.EncryptB2B(bucketKey, MasterKey)
	if err != nil {
		return err
	}
	// переводим в HEX
	bucketKeyHex := hex.EncodeToString(codedBucketkey)

	ble := len(bucketKeyHex)
	_ = ble

	order := "INSERT INTO USERA(username, password, bucketname, bucketkey, metadata) VALUES ($1, crypt($2, gen_salt('md5')), $1, $3, $4) ;"
	_, err = tx.Exec(ctx, order, userName, password, bucketKeyHex, metaData)
	if err != nil {
		return fmt.Errorf("add user error is %w", err)
	}
	return tx.Commit(ctx)
}

func (dataBase *DBstruct) CheckUserPassword(ctx context.Context, userName, password string) (yes bool) {
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
	db := dataBase.DB
	order := "SELECT userId from USERA WHERE username = $1 ;"
	row := db.QueryRow(ctx, order, userName) // password here - what was entered
	//var uId int
	err := row.Scan(&uId)
	return err == nil, uId
}
