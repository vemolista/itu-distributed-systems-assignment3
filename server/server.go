package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"slices"
	"sync"

	"github.com/vemolista/itu-distributed-systems-assignment3/v2/common"
	proto "github.com/vemolista/itu-distributed-systems-assignment3/v2/grpc"
	"google.golang.org/grpc"
)

type chitChatServer struct {
	proto.UnimplementedChitChatServer

	clock common.LamportClock

	mu            sync.Mutex
	activeClients []string
}

func (s *chitChatServer) Join(ctx context.Context, in *proto.JoinRequest) (*proto.JoinResponse, error) {
	s.clock.Update(in.LogicalTimestamp)

	s.mu.Lock()
	defer s.mu.Unlock()

	if slices.Contains(s.activeClients, in.Username) {
		return nil, fmt.Errorf("duplicate usernames not allowed")
	}

	s.activeClients = append(s.activeClients, in.Username)

	return &proto.JoinResponse{
		LogicalTimestamp: s.clock.Get(),
	}, nil
}

func (s *chitChatServer) Leave(ctx context.Context, in *proto.LeaveRequest) (*proto.LeaveResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !slices.Contains(s.activeClients, in.Username) {
		return nil, fmt.Errorf("no active client with username %s found", in.Username)
	}

	return &proto.LeaveResponse{}, nil
}

func (s *chitChatServer) SendMessage(ctx context.Context, in *proto.SendMessageRequest) (*proto.SendMessageResponse, error) {
	fmt.Printf("Received message from %s: '%s'\n", in.Message.Username, in.Message.Content)

	return nil, nil
}

func main() {
	server := &chitChatServer{activeClients: make([]string, 0), clock: common.NewLamportClock()}
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
