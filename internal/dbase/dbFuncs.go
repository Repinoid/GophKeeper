package dbase

import (
	"fmt"
	"strings"
	"time"

	"context"

	"crypto/rand"
	"encoding/hex"

	pb "gorsovet/cmd/proto"

	"gorsovet/internal/minio"
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
// PutToken to TOKENA table, with metadata
func (dataBase *DBstruct) PutToken(ctx context.Context, userName, token, metadata string) (err error) {
	userName = strings.ToUpper(userName)
	db := dataBase.DB

	order := "INSERT INTO TOKENA(userName, token, metadata) VALUES ($1, $2, $3) ;"
	_, err = db.Exec(ctx, order, userName, token, metadata)

	return
}

// 
func (dataBase *DBstruct) GetUserNameByToken(ctx context.Context, token string) (username string, err error) {
	order := "SELECT username from TOKENA WHERE token =  $1 ;"
	row := dataBase.DB.QueryRow(ctx, order, token)
	err = row.Scan(&username)
	return
}

func (dataBase *DBstruct) GetBucketKeyByUserName(ctx context.Context, username string) (bucketKey, bucketName string, err error) {
	order := "SELECT bucketkey, bucketname from USERA WHERE username =  $1 ;"
	row := dataBase.DB.QueryRow(ctx, order, strings.ToUpper(username))
	err = row.Scan(&bucketKey, &bucketName)
	return
}

func (dataBase *DBstruct) PutFileParams(ctx context.Context, object_id int32, username, fileURL, dataType, fileKey, metaData string, fileSize int32) (err error) {
	order := ""
	if object_id == 0 {
		order = "INSERT INTO DATAS AS args(userName, fileURL, dataType, fileKey, metaData, fileSize) VALUES ($1, $2, $3, $4, $5, $6) "
		// если для username файл fileURL уже существует, перезаписываем его, также ключ и тд.
		// ON CONFLICT потому что (username, fileURL) - Primary Key
		order += "ON CONFLICT (username, fileURL) DO UPDATE SET dataType=EXCLUDED.dataType, "
		order += "fileKey=EXCLUDED.fileKey, metaData=EXCLUDED.metaData, fileSize=EXCLUDED.filesize ;"
		_, err = dataBase.DB.Exec(ctx, order, username, fileURL, dataType, fileKey, metaData, fileSize)
	} else {
		// при обновлении записи надо заменить все её поля в строке БД за исключением номера.
		// И удалить прежний файл из бакета
		tx, err := dataBase.DB.Begin(ctx)
		if err != nil {
			return fmt.Errorf("error db.Begin  %w", err)
		}
		defer tx.Rollback(ctx)

		// определяем прежнее имя файла в S3
		order = "SELECT fileURL from DATAS WHERE username = $1 AND id = $2 ;"
		row := tx.QueryRow(ctx, order, username, object_id)
		urla := ""
		err = row.Scan(&urla)
		if err != nil {
			return fmt.Errorf("row.Scan(&urla) %w", err)
		}
		// получить имя бакета, может быть иным чем юзернейм
		bucketName := ""
		order = "SELECT bucketname from USERA WHERE username =  $1 ;"
		row = tx.QueryRow(ctx, order, username)
		err = row.Scan(&bucketName)
		if err != nil {
			return fmt.Errorf("row.Scan(&bucketname) %w", err)
		}
		// удалить файл в бакете
		err = minio.S3RemoveFile(ctx, models.MinioClient, bucketName, urla)
		if err != nil {
			models.Sugar.Debugf("bad S3RemoveFile %v", err)
			return err
		}
		// обновить запись, фактически оставив только её номер. всё остальное - новое, в т.ч. и тип
		order = "UPDATE DATAS SET fileURL=$1, dataType=$2, fileKey=$3, metaData=$4, filesize=$5 WHERE username=$6 AND id=$7 ;"
		_, err = tx.Exec(ctx, order, fileURL, dataType, fileKey, metaData, fileSize, username, object_id)
		if err != nil {
			return fmt.Errorf("UPDATE DATAS %w", err)
		}
		err = tx.Commit(ctx)
		if err != nil {
			return err
		}
	}
	if err != nil {
		models.Sugar.Debugf("PutFileParams %v\norder %s\n", err, order)
	}
	return
}

// RemoveObjects удаляет строку с id в таблице и возвращает имя файла, для удаления в S3
func (dataBase *DBstruct) RemoveObjects(ctx context.Context, username string, id int32) (fileURL string, err error) {

	tx, err := dataBase.DB.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("error db.Begin  %w", err)
	}
	defer tx.Rollback(ctx)

	order := "SELECT fileURL from DATAS WHERE username = $1 AND id = $2 ;"
	row := tx.QueryRow(ctx, order, username, id)
	urla := ""
	err = row.Scan(&urla)
	if err != nil {
		return "", fmt.Errorf("row.Scan(&urla) %w", err)
	}
	order = "DELETE from DATAS WHERE username = $1 AND id = $2 ;"
	_, err = tx.Exec(ctx, order, username, id)
	if err != nil {
		return "", fmt.Errorf("DELETE from DATAS %w", err)
	}
	return urla, tx.Commit(ctx)
}

// GetObjectsList list from DATAS table - список всех объектов юзера
func (dataBase *DBstruct) GetObjectsList(ctx context.Context, username string) (listing []*pb.ObjectParams, err error) {

	order := "SELECT id, fileURL, datatype, metadata, user_created_at, fileSize from DATAS WHERE username = $1 order by user_created_at ;"
	rows, err := dataBase.DB.Query(ctx, order, username) //
	if err != nil {
		models.Sugar.Debugf("db.Query %+v\n", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var pgTime time.Time
		ols := pb.ObjectParams{}
		err = rows.Scan(&ols.Id, &ols.Fileurl, &ols.DataType, &ols.Metadata, &pgTime, &ols.Size)
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

func (dataBase *DBstruct) GetObjectIdParams(ctx context.Context, username string, id int32) (param *pb.ObjectParams, err error) {
	obj := pb.ObjectParams{}

	order := "SELECT fileURL, filekey, datatype, metadata, filesize, user_created_at from DATAS WHERE username = $1 AND id = $2 ORDER BY user_created_at ;"
	row := dataBase.DB.QueryRow(ctx, order, username, id)
	// для скана времени - напрямую в струкруту не получается
	var pgTime time.Time
	err = row.Scan(&obj.Fileurl, &obj.Filekey, &obj.DataType, &obj.Metadata, &obj.Size, &pgTime)
	obj.CreatedAt = timestamppb.New(pgTime)
	if err != nil {
		models.Sugar.Debugf("row.Scan(&obj.Fileurl, %+v\n", err)
		return
	}
	return &obj, err
}
