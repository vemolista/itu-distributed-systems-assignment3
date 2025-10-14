package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/vemolista/itu-distributed-systems-assignment3/v2/common"
	proto "github.com/vemolista/itu-distributed-systems-assignment3/v2/grpc"
	"google.golang.org/grpc"
)

type chitChatServer struct {
	proto.UnimplementedChitChatServer

	clock common.LamportClock

	mu            sync.Mutex
	activeClients map[string]proto.ChitChat_ReceiveMessagesServer
}

func (s *chitChatServer) Join(ctx context.Context, in *proto.JoinRequest) (*proto.JoinResponse, error) {
	s.clock.Update(in.LogicalTimestamp)

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.activeClients[in.Username]; ok {
		return nil, fmt.Errorf("duplicate usernames not allowed")
	}

	return &proto.JoinResponse{
		LogicalTimestamp: s.clock.Get(),
	}, nil
}

func (s *chitChatServer) Leave(ctx context.Context, in *proto.LeaveRequest) (*proto.LeaveResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.activeClients[in.Username]; !ok {
		return nil, fmt.Errorf("no active client with username %s found", in.Username)
	}

	delete(s.activeClients, in.Username)

	return &proto.LeaveResponse{}, nil
}

func (s *chitChatServer) SendMessage(ctx context.Context, in *proto.SendMessageRequest) (*proto.SendMessageResponse, error) {
	fmt.Printf("Received message from %s: '%s'\n", in.Message.Username, in.Message.Content)

	response := &proto.ReceiveMessagesResponse{
		Message:          in.Message,
		LogicalTimestamp: 1, // TODO
	}

	fmt.Printf("%v\n", len(s.activeClients))
	for k := range s.activeClients {
		fmt.Println(k)
	}

	s.mu.Lock()
	for k, stream := range s.activeClients {
		if k == in.Message.Username {
			continue
		}

		if err := stream.Send(response); err != nil {
			log.Printf("failed to send message to client '%s': %v", k, err)
		}
	}
	s.mu.Unlock()

	return &proto.SendMessageResponse{
		LogicalTimestamp: 1, // TODO
	}, nil
}

func (s *chitChatServer) ReceiveMessages(in *proto.ReceiveMessagesRequest, stream proto.ChitChat_ReceiveMessagesServer) error {
	s.mu.Lock()
	fmt.Printf("%s", in.Username)
	fmt.Printf("%v\n", len(s.activeClients))
	s.activeClients[in.Username] = stream
	fmt.Printf("%v\n", len(s.activeClients))
	s.mu.Unlock()

	<-stream.Context().Done()

	s.mu.Lock()
	delete(s.activeClients, in.Username)
	s.mu.Unlock()

	return nil
}

func main() {
	server := &chitChatServer{
		activeClients: make(map[string]proto.ChitChat_ReceiveMessagesServer, 0),
		clock:         common.NewLamportClock(),
	}
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
