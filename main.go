package main

import (
	"fmt"
	pb "github.com/aau-network-security/haaukins-store/proto"
	rpc "github.com/aau-network-security/haaukins-store/grpc"
	_ "github.com/lib/pq"
	"log"
	"net"
)

func main() {

	s := rpc.InitilizegRPCServer()
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	opts, err := s.GrpcOpts()
	if err != nil {
		log.Fatal("failed to retrieve server options")
	}

	gRPCServer := s.GetGRPCServer(opts...)
	pb.RegisterStoreServer(gRPCServer, s)
	fmt.Println("waiting client")
	if err := gRPCServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

}
