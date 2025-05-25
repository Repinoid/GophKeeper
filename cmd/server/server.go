package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	pb "gorsovet/cmd/proto"

	"google.golang.org/grpc"
)

type GkeeperService struct {
	pb.UnimplementedGkeeperServer
}

func main() {
	if err := Run(); err != nil {
		log.Printf("Server Shutdown by syscall, ListenAndServe message -  %v\n", err)
	}
}

func Run() (err error) {

	// во первЫх строках - если сеть не прослушивается, до дальше и делать нечего
	listen, err := net.Listen("tcp", ":3200")
	if err != nil {
		log.Fatal(err)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterGkeeperServer(grpcServer, &GkeeperService{})

	// Graceful shutdown channel
	done := make(chan bool, 1)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("Server is shutting down...")

		// Give existing connections 30 seconds to complete
		_, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Stop accepting new connections and wait for existing ones
		grpcServer.GracefulStop()
		// Alternatively, use s.Stop() for immediate shutdown
		close(done)
	}()
	
	if err := grpcServer.Serve(listen); err != nil {
		log.Fatal(err)
	}

	<-done
	log.Println("Server stopped")

	return
}
