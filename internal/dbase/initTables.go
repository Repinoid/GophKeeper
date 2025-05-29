package dbase

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DBstruct struct {
	//	DB     *pgx.Conn
	DB *pgxpool.Pool
}

// ConnectToDB получить эндпоинт Базы Данных
func ConnectToDB(ctx context.Context, DBEndPoint string) (dataBase *DBstruct, err error) {

	//	baza, err := pgx.Connect(ctx, DBEndPoint)
	baza, err := pgxpool.New(ctx, DBEndPoint)
	if err != nil {
		return nil, fmt.Errorf("can't connect to DB %s err %w", DBEndPoint, err)
	}
	dataBase = &DBstruct{DB: baza} // Initialize

	return
}

// UsersTableCreation создание таблицы юзеров
func (dataBase *DBstruct) UsersTableCreation(ctx context.Context) error {

	db := dataBase.DB
	_, err := db.Exec(ctx, "CREATE EXTENSION IF NOT EXISTS pgcrypto;") // расширение для хэширования паролей
	if err != nil {
		return fmt.Errorf("error CREATE EXTENSION pgcrypto; %w", err)
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
		return fmt.Errorf("create USERS table. %w", err)
	}
	models.Sugar.Debugln("USERA table is created")
	return nil
}

func (dataBase *DBstruct) TokensTableCreation(ctx context.Context) error {
	db := dataBase.DB
	creatorOrder :=
		"CREATE TABLE IF NOT EXISTS TOKENA" +
			"(id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY, " +
			"username VARCHAR(64) NOT NULL, " +
			"token TEXT NOT NULL, " +
			"metadata TEXT, " +
			"token_created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP);"

	_, err := db.Exec(ctx, creatorOrder)
	if err != nil {
		return fmt.Errorf("create TOKENS table. %w", err)
	}
	return nil
}

func (dataBase *DBstruct) DataTableCreation(ctx context.Context) error {

	db := dataBase.DB

	creatorOrder :=
		"CREATE TABLE IF NOT EXISTS DATAS" +
			"(id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY, " +
			"username VARCHAR(64) NOT NULL, " +
			"fileURL TEXT NOT NULL, " +
			"datatype VARCHAR(20) NOT NULL, " +
			"fileKey TEXT NOT NULL, " +
			"fileSize int NOT NULL, " +
			"metadata TEXT, " +
			"user_created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP);"

	_, err := db.Exec(ctx, creatorOrder)
	if err != nil {
		return fmt.Errorf("create DATA table. %w", err)
	}
	return nil
}

func (dataBase *DBstruct) CloseBase() {
	dataBase.DB.Close()
}
