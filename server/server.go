package main

import (
	"context"
	"fmt"
	"log"
	"net"

	proto "github.com/vemolista/itu-distributed-systems-assignment3/v2/grpc"
	"google.golang.org/grpc"
)

type chitChatServer struct {
	proto.UnimplementedChitChatServer
}

func (s *chitChatServer) BroadcastMessage(ctx context.Context, in *proto.BroadcastRequest) (*proto.BroadcastResponse, error) {
	fmt.Println("Broadcasting message.")

	return nil, nil
}

func main() {
	server := &chitChatServer{}
	server.start()
}

func (s *chitChatServer) start() {
	server := grpc.NewServer()

	network := "tcp"
	port := ":5050"

	listener, err := net.Listen(network, port)
	if err != nil {
		log.Fatalf("failed to create a %s listener on port %s: %v\n", network, port, err)
	}

	proto.RegisterChitChatServer(server, s)

	if err := server.Serve(listener); err != nil {
		log.Fatalf("failed to start serving requests: %v", err)
	}
}
