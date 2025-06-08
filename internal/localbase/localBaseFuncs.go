package localbase

import (
	"fmt"
	"gorsovet/internal/models"
	"os"
	"strings"
	"time"

	pb "gorsovet/cmd/proto"

	"google.golang.org/protobuf/types/known/timestamppb"
	_ "modernc.org/sqlite"
)

// AddUser вносит в локальную БД данные о зарегистрированных пользователях, запускается в registerFlagFunc после внесения того же на БД сервера
func AddUser(localsql LocalDB, username, password, metaFlag string) (err error) {

	password, err = hashPassword(password)
	if err != nil {
		return
	}
	_, err = localsql.SQLdb.Exec("INSERT INTO USERA(username, password, bucketname, metadata) VALUES(?,?,?,?)",
		strings.ToUpper(username), password, strings.ToLower(username), metaFlag)
	if err != nil {
		fmt.Printf("error table USERA insert  %[1]v\n", err)
		return
	}

	bucket := models.LocalS3Dir + "/" + strings.ToLower(username)
	// create  S3 bucket for user if not exists
	if _, err := os.Stat(bucket); os.IsNotExist(err) {
		// Create the directory with 0755 permissions (rwx for owner, rx for group/others)
		err := os.Mkdir(bucket, 0755)
		if err != nil {
			return err
		}
	}
	return
}

func Login(localsql LocalDB, username, password string) (err error) {
	order := "SELECT password from USERA WHERE username = ? ;"
	row := localsql.SQLdb.QueryRow(order, strings.ToUpper(username))
	var passHash = ""
	err = row.Scan(&passHash)
	if err != nil {
		models.Sugar.Debug(err)
		return
	}
	err = checkPassword(passHash, password)
	return
}

// PutFileParams - внесение данных о записи в локальную БД, запускается после подобного для БД сервера
func PutFileParams(localsql LocalDB, object_id int32, username, fileURL, dataType, metaData string) (err error) {
	username = strings.ToUpper(username)
	order := ""
	if object_id == 0 {
		order = "INSERT INTO DATAS AS args(userName, fileURL, dataType, metaData) VALUES (?, ?, ?, ?) ;"
		// если для username файл fileURL уже существует, перезаписываем его, также ключ и тд.
		// ON CONFLICT потому что (username, fileURL) - Primary Key
		//order += "ON CONFLICT (username, fileURL) DO UPDATE SET dataType=EXCLUDED.dataType, metaData=EXCLUDED.metaData ;"

		_, err = localsql.SQLdb.Exec(order, username, fileURL, dataType, metaData)
	} else {
		// при обновлении записи надо заменить все её поля в строке БД за исключением номера.
		// И удалить прежний файл из бакета
		tx, err := localsql.SQLdb.Begin()
		if err != nil {
			return fmt.Errorf("error db.Begin  %w", err)
		}
		defer tx.Rollback()

		// определяем прежнее имя файла в S3
		order = "SELECT fileURL from DATAS WHERE username = ? AND id = ? ;"
		row := tx.QueryRow(order, username, object_id)
		urla := ""
		err = row.Scan(&urla)
		if err != nil {
			return fmt.Errorf("row.Scan(&urla) %w", err)
		}
		// получить имя бакета, может быть иным чем юзернейм
		bucketName := ""
		order = "SELECT bucketname from USERA WHERE username =  ? ;"
		row = tx.QueryRow(order, username)
		err = row.Scan(&bucketName)
		if err != nil {
			return fmt.Errorf("row.Scan(&bucketname) %w", err)
		}
		// удалить файл в бакете
		fnam := models.LocalS3Dir + "/" + strings.ToLower(username) + "/" + urla
		err = os.Remove(fnam)
		if err != nil {
			models.Sugar.Debugf("bad local S3RemoveFile %v", err)
			return err // сработает также defer tx.Rollback(ctx)
		}
		// обновить запись, фактически оставив только её номер. всё остальное - новое, в т.ч. и тип
		order = "UPDATE DATAS SET fileURL=?, dataType=?, metaData=? WHERE username=? AND id=? ;"
		_, err = tx.Exec(order, fileURL, dataType, metaData, username, object_id)
		if err != nil {
			return fmt.Errorf("UPDATE DATAS %w", err)
		}

		err = tx.Commit()
		if err != nil {
			return err
		}
	}
	if err != nil {
		models.Sugar.Debugf("PutFileParams %v\norder %s\n", err, order)
	}
	return
}

// GetList - получение списка записей из локальной БД, срабатывает при недоступности Сервера
func GetList(localsql LocalDB, username string) (listing []*pb.ObjectParams, err error) {
	order := "SELECT id, fileURL, datatype, metadata, user_created_at from DATAS WHERE username = ? order by user_created_at ;"
	rows, err := localsql.SQLdb.Query(order, username) //
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

// GetList - получение параметров записи id из локальной БД, срабатывает при недоступности Сервера
func GetRecordHead(localsql LocalDB, id int32) (head *pb.ObjectParams, err error) {

	order := "SELECT username, fileURL, datatype, metadata, user_created_at from DATAS WHERE id = ? ;"
	row := localsql.SQLdb.QueryRow(order, id) //
	var pgTime time.Time
	ols := pb.ObjectParams{}
	// filekey - for username here
	err = row.Scan(&ols.Filekey, &ols.Fileurl, &ols.DataType, &ols.Metadata, &pgTime)
	if err != nil {
		models.Sugar.Debug(err)
		return
	}
	ols.CreatedAt = timestamppb.New(pgTime)
	head = &ols
	return
}

func Remover(localsql LocalDB, id int32) (err error) {
	// добываем пусть к файлу из DATAS
	order := "SELECT username, fileURL from DATAS WHERE id = ? ;"
	row := localsql.SQLdb.QueryRow(order, id) //
	var folder, filename string
	// filekey - for username here
	err = row.Scan(&folder, &filename)
	if err != nil {
		models.Sugar.Debug(err)
		return
	}
	// удаляем файл с данными
	fnam := models.LocalS3Dir + "/" + strings.ToLower(folder) + "/" + filename
	err = os.Remove(fnam)
	if err != nil {
		models.Sugar.Debug(err)
		return
	}
	// удаляем запись в таблице локальной БД
	order = "DELETE from DATAS WHERE id = ? ;"
	_, err = localsql.SQLdb.Exec(order, id)
	if err != nil {
		models.Sugar.Debug(err)
		return
	}

	return
}
