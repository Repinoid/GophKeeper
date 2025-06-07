package main

import (
	"fmt"
	"strings"

	_ "modernc.org/sqlite"
)

func main() {

	localBase, err := ConnectToLocalDB("file:test.db")
	if err != nil {
		fmt.Printf("error ConnectToLocalDB  %v", err)
		return
	}

	err = localBase.UsersTableCreation()
	if err != nil {
		fmt.Printf("error UsersTableCreation  %v", err)
		return
	}

	err = localBase.TokensTableCreation()
	if err != nil {
		fmt.Printf("error TokensTableCreation  %v", err)
		return
	}
	err = localBase.DataTableCreation()
	if err != nil {
		fmt.Printf("error DataTableCreation  %v", err)
		return
	}

	username := "usern"
	password := "pass"

	password, err = hashPassword(password)
	if err != nil {
		fmt.Printf("error hash  %v", err)
		return
	}

	_, err = localBase.SQLdb.Exec("INSERT INTO USERA(username, password, bucketname, metadata) VALUES(?,?,?,?)",
		strings.ToUpper(username), password, strings.ToLower(username), "meta")
	if err != nil {
		fmt.Printf("error table USERA insert  %[1]v\n", err)
		return
	}
	_, err = localBase.SQLdb.Exec("DELETE FROM USERA WHERE username=?",
		strings.ToUpper(username))
	if err != nil {
		fmt.Printf("error table USERA delete  %[1]v\n", err)
		return
	}

	localBase.CloseBase()

}
