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

var ()

// UsersTableCreation создание таблицы юзеров
func (dataBase *DBstruct) UsersTableCreation(ctx context.Context) error {

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
