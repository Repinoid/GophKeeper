package main

import (
	"context"
	"log"

	pb "gorsovet/cmd/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var gPort = ":3200"

func main() {
	ctx := context.Background()
	err := initClient(ctx)
	if err != nil {
		log.Fatal(err)
	}

	if err := run(ctx); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) (err error) {

	conn, err := grpc.NewClient(gPort, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	client := pb.NewGkeeperClient(conn)
	req := &pb.LoginRequest{Username: "n77", Password: "passw"}
	resp, err := client.LoginUser(ctx, req)
	log.Printf("%+v", resp)

	token := resp.GetToken()

	reqtxt := &pb.PutTextRequest{Token: token, Textdata: "12345", Metadata: "metta"}
	respt, err := client.PutText(ctx, reqtxt)
	log.Printf("%+v", respt)

	return
}
