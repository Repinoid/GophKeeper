package localbase

import (
	"database/sql"
	"fmt"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

type localDB struct {
	//	DB     *pgx.Conn
	SQLdb *sql.DB
}

// ConnectToDB получить эндпоинт Базы Данных
func ConnectToLocalDB(DBEndPoint string) (dataBase *localDB, err error) {
	db, err := sql.Open("sqlite", DBEndPoint)
	if err != nil {
		return
	}
	dataBase = &localDB{SQLdb: db}
	return
}

func (dataBase *localDB) UsersTableCreation() (err error) {

	db := dataBase.SQLdb
	// _, err := db.Exec("CREATE EXTENSION IF NOT EXISTS pgcrypto;") // расширение для хэширования паролей
	// if err != nil {
	// 	return fmt.Errorf("error CREATE EXTENSION pgcrypto; %w", err)
	// }

	creatorOrder :=
		"CREATE TABLE IF NOT EXISTS USERA" +
			"(userId INTEGER PRIMARY KEY AUTOINCREMENT, " +
			"username VARCHAR(64) UNIQUE, " +
			"password TEXT NOT NULL, " +
			"bucketname VARCHAR(64) NOT NULL, " +
			"metadata TEXT, " +
			"user_created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP);"

	_, err = db.Exec(creatorOrder)
	if err != nil {
		return fmt.Errorf("create USERS table. %w", err)
	}
	//	models.Sugar.Debugln("USERA table is created")
	return nil
}

func (dataBase *localDB) TokensTableCreation() error {
	db := dataBase.SQLdb
	creatorOrder :=
		"CREATE TABLE IF NOT EXISTS TOKENA" +
			"(id INTEGER PRIMARY KEY AUTOINCREMENT, " +
			"username VARCHAR(64) NOT NULL, " +
			"token TEXT NOT NULL, " +
			"metadata TEXT, " +
			"token_created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP);"

	_, err := db.Exec(creatorOrder)
	if err != nil {
		return fmt.Errorf("create TOKENS table. %w", err)
	}
	return nil
}

func (dataBase *localDB) DataTableCreation() error {

	db := dataBase.SQLdb

	creatorOrder :=
		"CREATE TABLE IF NOT EXISTS DATAS" +
			"(id INTEGER  PRIMARY KEY AUTOINCREMENT, " +
			"username VARCHAR(64) NOT NULL, " +
			"fileURL TEXT NOT NULL, " +
			"datatype VARCHAR(20) NOT NULL, " +
			"fileKey TEXT NOT NULL, " +
			"fileSize int NOT NULL, " +
			"metadata TEXT, " +
			"user_created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP) ; " //+
		// имя юзера и имя файла должны быть уникальными - в одной корзине нет одинаковых имён файлов
		//	"PRIMARY KEY (username, fileURL) ) WITHOUT ROWID ;"

	_, err := db.Exec(creatorOrder)
	if err != nil {
		return fmt.Errorf("create DATA table. %w", err)
	}
	return nil
}

func (dataBase *localDB) CloseBase() {
	dataBase.SQLdb.Close()
}

func hashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hashedPassword), nil
}

func checkPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}
