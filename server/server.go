package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
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
	s.clock.Update(in.LogicalTimestamp)

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.activeClients[in.Username]; !ok {
		return nil, fmt.Errorf("no active client with username %s found", in.Username)
	}

	delete(s.activeClients, in.Username)

	return &proto.LeaveResponse{}, nil
}

func (s *chitChatServer) SendMessage(ctx context.Context, in *proto.SendMessageRequest) (*proto.SendMessageResponse, error) {
	s.clock.Update(in.LogicalTimestamp)

	response := &proto.ReceiveMessagesResponse{
		Message:          in.Message,
		LogicalTimestamp: s.clock.Get(),
	}

	s.mu.Lock()
	for username, stream := range s.activeClients {
		if username == in.Message.Username {
			continue
		}

		if err := stream.Send(response); err != nil {
			log.Printf("failed to send message to client '%s': %v", username, err)
		}
	}
	s.mu.Unlock()

	return &proto.SendMessageResponse{
		LogicalTimestamp: s.clock.Get(),
	}, nil
}

func (s *chitChatServer) ReceiveMessages(in *proto.ReceiveMessagesRequest, stream proto.ChitChat_ReceiveMessagesServer) error {
	s.clock.Update(in.LogicalTimestamp)

	s.mu.Lock()
	s.activeClients[in.Username] = stream
	s.mu.Unlock()

	s.SendMessage(context.Background(), &proto.SendMessageRequest{
		Message: &proto.ChatMessage{
			Type:    proto.MessageType_SYSTEM_JOIN,
			Content: fmt.Sprintf("Participant %s has joined Chit Chat at logical time %d", in.Username, s.clock.Get()),
		},
		LogicalTimestamp: s.clock.Get(),
	})

	<-stream.Context().Done()

	s.mu.Lock()
	delete(s.activeClients, in.Username)
	s.mu.Unlock()

	s.clock.Increment()

	s.SendMessage(context.Background(), &proto.SendMessageRequest{
		Message: &proto.ChatMessage{
			Type:    proto.MessageType_SYSTEM_LEAVE,
			Content: fmt.Sprintf("Participant %s has left Chit Chat at logical time %d", in.Username, s.clock.Get()),
		},
		LogicalTimestamp: s.clock.Get(),
	})

	return nil
}

func main() {
	server := &chitChatServer{
		activeClients: make(map[string]proto.ChitChat_ReceiveMessagesServer, 0),
		clock:         common.NewLamportClock(),
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for sig := range c {
			log.Printf("server shutdown: %s (timestamp: %d)\n", sig.String(), server.clock.Get())
			fmt.Printf("server shutdown: %s (timestamp: %d)\n", sig.String(), server.clock.Get())

			os.Exit(0)
		}
	}()

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
