package localbase

import (
	"fmt"
	"gorsovet/internal/models"
	"os"
	"strings"

	_ "modernc.org/sqlite"
)

func AddUser(localsql LocalDB, username, password string) (err error) {

	password, err = hashPassword(password)
	if err != nil {
		return
	}

	_, err = localsql.SQLdb.Exec("INSERT INTO USERA(username, password, bucketname) VALUES(?,?,?)",
		strings.ToUpper(username), password, strings.ToLower(username))
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

