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
	"gorsovet/internal/privacy"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"google.golang.org/grpc/health"
)

func main() {

	ctx := context.Background()
	err := initServer(ctx)
	if err != nil {
		models.Sugar.Fatalf("The Server could not start. Reason - %v", err)
	}

	if err = Run(ctx); err != nil {
		log.Printf("Server Shutdown by syscall, ListenAndServe message -  %v\n", err)
	}
}

func Run(ctx context.Context) (err error) {

	// во первЫх строках - если сеть не прослушивается, до дальше и делать нечего
	listen, err := net.Listen("tcp", models.Gport)
	if err != nil {
		log.Fatal(err)
	}
	// сертификаты в папке tls, для сервера GRPC требует и публичный и приватный
	creds, err := privacy.LoadTLSCredentials("../tls/public.crt", "../tls/private.key")
	//	creds, err := privacy.LoadTLSCredentials("../tls/cert.pem", "../tls/key.pem")
	if err != nil {
		log.Fatalf("failed to load TLS credentials: %v", err)
	}
	grpcServer := grpc.NewServer(grpc.Creds(creds))

	healthServer := health.NewServer()
	// Устанавливаем статус SERVING
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	// Регистрируем health сервис
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)

	//	grpcServer := grpc.NewServer()

	pb.RegisterGkeeperServer(grpcServer, &handlers.GkeeperService{})
	// reflection nice for grpcurl
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

		//grpcServer.Stop()
		grpcServer.GracefulStop()
		// Alternatively, use s.Stop() for immediate shutdown
		close(done)
	}()

	log.Println("GRPC Server Started")
	// если сервер не стартует, до дальше и делать нечего
	if err := grpcServer.Serve(listen); err != nil {
		log.Fatal(err)
	}
	<-done
	log.Println("Server stopped")

	return
}

//  openssl req -x509 -newkey rsa:4096 -keyout private.key -out public.crt -days 365 -nodes -subj "/CN=localhost"   -addext "subjectAltName=DNS:localhost,IP:127.0.0.1"
