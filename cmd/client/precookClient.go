package main

import (
	"context"
	"errors"
	"flag"
	"gorsovet/internal/models"
	"log"

	"go.uber.org/zap"
)

var metaFlag, registerFlag, loginFlag, putFileFlag, putTextFlag  string
var removeFlag, getFileFlag int
var listFlag bool

func initClient(ctx context.Context) (err error) {

	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal("cannot initialize zap")
	}
	defer logger.Sync()
	models.Sugar = *logger.Sugar()

	flag.StringVar(&metaFlag, "meta", "", "metadata, -meta=\"...metadata text...\"")
	flag.StringVar(&registerFlag, "register", "", "register new user, -register=\"userName,password\" divided by comma")
	flag.StringVar(&loginFlag, "login", "", "login to Server, -login=\"userName,password\" divided by comma")

	flag.StringVar(&putFileFlag, "putfile", "", "put file to storage, -putfile=\"filePath/filename\"")
	flag.StringVar(&putTextFlag, "puttext", "", "put text to storage, -puttext=\"... your text ...\"")

	flag.IntVar(&getFileFlag, "getfile", 0, "get record to storage, -getfile=<id of record>, take it by -list")
	
	flag.BoolVar(&listFlag, "list", false, "list objects, -list")
	flag.IntVar(&removeFlag, "remove", 0, "remove object, -remove=object_id ")
	flag.Parse()

	if metaFlag == "" && registerFlag == "" && loginFlag == "" && putFileFlag == "" &&
		putTextFlag == "" && getFileFlag == 0 && !listFlag && removeFlag == 0 {
		return errors.New("no any flag")
	}

	return
}
