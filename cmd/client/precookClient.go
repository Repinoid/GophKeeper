package main

import (
	"context"
	"errors"
	"flag"
)

var metaFlag, registerFlag, loginFlag, putFileFlag, putTextFlag, fnameFlag, putCardFlag string
var removeFlag, getFileFlag, updateFlag, showFlag int
var listFlag bool

func initGrpcClient(ctx context.Context) (err error) {

	flag.StringVar(&metaFlag, "meta", "", "metadata, -meta=\"...metadata text...\"")
	flag.StringVar(&registerFlag, "register", "", "register new user, -register=\"userName,password\" divided by comma")
	flag.StringVar(&loginFlag, "login", "", "login to Server, -login=\"userName,password\" divided by comma")

	flag.StringVar(&putFileFlag, "putfile", "", "put file to storage, -putfile=\"filePath/filename\"")
	flag.StringVar(&putTextFlag, "puttext", "", "put text to storage, -puttext=\"... your text ...\"")
	flag.StringVar(&putCardFlag, "putcard", "", "put card to storage, -putcard=\"cardnumber digits,expiration MM/YY, CSV, cardholder name\"")

	flag.BoolVar(&listFlag, "list", false, "list objects, -list")
	flag.IntVar(&showFlag, "show", 0, "show record parameters from storage, -show=<id of record>, take it by -list")
	flag.IntVar(&getFileFlag, "get", 0, "download record to file, -get=<id of record> -file=\"filePath/filename\", if no -file flag - filename from storage")
	flag.StringVar(&fnameFlag, "file", "", "file name to save file, -file=\"filePath/filename\", uses only with -get")

	flag.IntVar(&removeFlag, "remove", 0, "remove object, -remove=object_id ")
	flag.IntVar(&updateFlag, "update", 0, "update object, use with -put* -update=object_id, use with -put*")
	flag.Parse()

	// metaFlag fnameFlag updateFlag используются только совместно
	// проверка на наличие флагов, в client.go срабатывает первый ненулевой
	if registerFlag == "" && loginFlag == "" && putTextFlag == "" && putFileFlag == "" && !listFlag && removeFlag == 0 && showFlag == 0 &&
		getFileFlag == 0 && putCardFlag == "" {
		return errors.New("no any flag ")
	}
	// local bases init

	return
}
func initLocalClient(ctx context.Context) (err error) {

	flag.StringVar(&loginFlag, "login", "", "login to Server, -login=\"userName,password\" divided by comma")

	flag.BoolVar(&listFlag, "list", false, "list objects, -list")
	flag.IntVar(&showFlag, "show", 0, "show record parameters from storage, -show=<id of record>, take it by -list")
	flag.IntVar(&getFileFlag, "get", 0, "download record to file, -get=<id of record> -file=\"filePath/filename\", if no -file flag - filename from storage")
	flag.StringVar(&fnameFlag, "file", "", "file name to save file, -file=\"filePath/filename\", uses only with -get")

	flag.Parse()

	// metaFlag fnameFlag updateFlag используются только совместно
	// проверка на наличие флагов, в client.go срабатывает первый ненулевой
	if loginFlag == "" && !listFlag && showFlag == 0 && getFileFlag == 0 {
		return errors.New("no any flag in local mode")
	}
	// local bases init

	return
}
