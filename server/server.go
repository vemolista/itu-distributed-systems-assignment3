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

	clock  common.LamportClock
	logger *log.Logger

	mu            sync.Mutex
	activeClients map[string]proto.ChitChat_ReceiveMessagesServer
}

func (s *chitChatServer) Join(ctx context.Context, in *proto.JoinRequest) (*proto.JoinResponse, error) {
	s.clock.Update(in.LogicalTimestamp)

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.activeClients[in.Username]; ok {
		msg := "duplicate usernames not allowed"
		return nil, fmt.Errorf("%s", msg)
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
		msg := fmt.Sprintf("no active client with username %s found", in.Username)
		return nil, fmt.Errorf("%s", msg)
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

	// Determine event type for logging
	eventType := "USER_MESSAGE"
	if in.Message.Type == proto.MessageType_SYSTEM_JOIN {
		eventType = "SYSTEM_JOIN"
	} else if in.Message.Type == proto.MessageType_SYSTEM_LEAVE {
		eventType = "SYSTEM_LEAVE"
	}

	s.mu.Lock()
	for username, stream := range s.activeClients {
		if username == in.Message.Username {
			continue
		}

		if err := stream.Send(response); err != nil {
			s.logger.Printf("[component: server] [client: %s] [event: SEND_ERROR] [timestamp: %d] [error: %v]",
				username, s.clock.Get(), err)
		} else {
			// Log successful message broadcast to each client
			s.logger.Printf("[component: server] [client: %s] [event: %s] [timestamp: %d] [content: %s]",
				username, eventType, s.clock.Get(), in.Message.Content)
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

	// Log broadcasting join message
	s.logger.Printf("[component: server] [client: %s] [event: BROADCAST_JOIN] [timestamp: %d] [content: broadcasting join message]",
		in.Username, s.clock.Get())

	s.SendMessage(context.Background(), &proto.SendMessageRequest{
		Message: &proto.ChatMessage{
			Username: in.Username, // Set username so the sender doesn't receive their own join message
			Type:     proto.MessageType_SYSTEM_JOIN,
			Content:  fmt.Sprintf("Participant %s has joined Chit Chat at logical time %d", in.Username, s.clock.Get()),
		},
		LogicalTimestamp: s.clock.Get(),
	})

	<-stream.Context().Done()

	s.clock.Increment()

	s.mu.Lock()
	delete(s.activeClients, in.Username)
	s.mu.Unlock()

	// Log broadcasting leave message
	s.logger.Printf("[component: server] [client: %s] [event: BROADCAST_LEAVE] [timestamp: %d] [content: broadcasting leave message]",
		in.Username, s.clock.Get())

	s.SendMessage(context.Background(), &proto.SendMessageRequest{
		Message: &proto.ChatMessage{
			Username: in.Username, // Set username so the sender doesn't receive their own leave message
			Type:     proto.MessageType_SYSTEM_LEAVE,
			Content:  fmt.Sprintf("Participant %s has left Chit Chat at logical time %d", in.Username, s.clock.Get()),
		},
		LogicalTimestamp: s.clock.Get(),
	})

	return nil
}

func main() {
	file, err := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("failed to open log file: %v\n", err)
	}
	logger := log.New(file, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)

	server := &chitChatServer{
		activeClients: make(map[string]proto.ChitChat_ReceiveMessagesServer, 0),
		clock:         common.NewLamportClock(),
		logger:        logger,
	}

	grpcServer := grpc.NewServer()
	proto.RegisterChitChatServer(grpcServer, server)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for sig := range c {
			// Log server shutdown
			logger.Printf("[component: server] [event: SHUTDOWN] [timestamp: %d] [signal: %s]",
				server.clock.Get(), sig.String())
			fmt.Printf("server shutdown: %s (timestamp: %d)\n", sig.String(), server.clock.Get())
			os.Exit(0)
		}
	}()

	server.start(logger)

}

func (s *chitChatServer) start(logger *log.Logger) {

	server := grpc.NewServer()

	network := "tcp"
	port := ":5050"

	listener, err := net.Listen(network, port)
	if err != nil {
		logger.Fatalf("failed to create a %s listener on port %s: %v\n", network, port, err)
	}

	proto.RegisterChitChatServer(server, s)

	// Log server start
	s.logger.Printf("[component: server] [event: START] [timestamp: %d] [address: %s%s]",
		s.clock.Get(), network, port)
	log.Printf("server started on %s%s (timestamp: %d)\n", network, port, s.clock.Get())

	if err := server.Serve(listener); err != nil {
		logger.Fatalf("failed to start serving requests: %v", err)
	}
}
