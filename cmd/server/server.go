package main

import (
	"log"
	"net"

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

	//	var wg sync.WaitGroup
	//ctx, cancel := context.WithCancel(context.Background())

	// go func() {
	 	exit := make(chan os.Signal, 1)
	 	signal.Notify(exit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	// 	<-exit
	// 	cancel()
	// }()
	// gRPC server, default port ":3200" or from parameters
	listen, err := net.Listen("tcp", ":3200")
	if err != nil {
		log.Fatal(err)
	}

	grpcServer := grpc.NewServer()

	pb.RegisterGkeeperServer(grpcServer, &GkeeperService{})

	//	wg.Add(1)
	if err := grpcServer.Serve(listen); err != nil {
		log.Fatal(err)
	}

	// go func() {
	// 	<-ctx.Done()
	// 	//		defer wg.Done()

	// 	// GracefulStop stops the gRPC server gracefully.
	// 	// It stops the server from accepting new connections and RPCs and blocks until all the pending RPCs are finished.
	// 	grpcServer.GracefulStop()
	// 	// defer для порядку
	// 	defer fmt.Println("Сервер gRPC Shutdown gracefully")
	// }()
	//	wg.Wait()
	return
}
