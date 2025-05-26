package main

import (
	"context"
	"flag"
)

func initClient(ctx context.Context) (err error) {

	var metaFlag, registerFlag, loginFlag string

	flag.StringVar(&metaFlag, "m", "", "metadata, -m=\"...metadata text...\"")
	flag.StringVar(&registerFlag, "r", "", "register new user, -r=\"userName,password,metadata\" divided by commas")
	flag.StringVar(&loginFlag, "l", "", "login to Server, -l=\"userName,password\" divided by comma")
	flag.Parse()

	return
}
