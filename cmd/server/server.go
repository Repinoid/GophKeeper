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
	"gorsovet/internal/handlers"
	"gorsovet/internal/models"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic("cannot initialize zap")
	}
	defer logger.Sync()
	models.Sugar = *logger.Sugar()

	if err := Run(); err != nil {
		log.Printf("Server Shutdown by syscall, ListenAndServe message -  %v\n", err)
	}
}

func Run() (err error) {

	// во первЫх строках - если сеть не прослушивается, до дальше и делать нечего
	listen, err := net.Listen("tcp", models.Gport)
	if err != nil {
		log.Fatal(err)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterGkeeperServer(grpcServer, &handlers.GkeeperService{})
	// gRPC server with reflection enabled, which will allow tools like grpcurl to inspect
	// your service without needing proto files.
	reflection.Register(grpcServer)

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

	log.Println("Server UP")
	if err := grpcServer.Serve(listen); err != nil {
		log.Fatal(err)
	}
	<-done
	log.Println("Server stopped")

	return
}
