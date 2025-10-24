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
	"github.com/vemolista/itu-distributed-systems-assignment3/v2/common/logging"
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
	logging.Log(logging.Server{}, "join", "join request received", s.clock.Get())

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.activeClients[in.Username]; ok {
		msg := "duplicate usernames not allowed"

		logging.Log(logging.Server{}, "join", msg, s.clock.Get())
		return nil, fmt.Errorf("%s", msg)
	}

	return &proto.JoinResponse{
		LogicalTimestamp: s.clock.Get(),
	}, nil
}

func (s *chitChatServer) Leave(ctx context.Context, in *proto.LeaveRequest) (*proto.LeaveResponse, error) {
	s.clock.Update(in.LogicalTimestamp)
	logging.Log(logging.Server{}, "leave", "leave request received", s.clock.Get())

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.activeClients[in.Username]; !ok {
		msg := fmt.Sprintf("no active client with username %s found", in.Username)
		logging.Log(logging.Server{}, "leave", msg, s.clock.Get())

		return nil, fmt.Errorf("%s", msg)
	}

	delete(s.activeClients, in.Username)

	return &proto.LeaveResponse{}, nil
}

func (s *chitChatServer) SendMessage(ctx context.Context, in *proto.SendMessageRequest) (*proto.SendMessageResponse, error) {
	s.clock.Update(in.LogicalTimestamp)
	msg := fmt.Sprintf("send message request received, content: '%s', type: %s", in.Message.Content, in.Message.Type.String())
	logging.Log(logging.Server{}, "send message", msg, s.clock.Get())

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
			msg := fmt.Sprintf("failed to send message to client '%s': %v", username, err)
			logging.Log(logging.Server{}, "send message", msg, s.clock.Get())
		}
	}
	s.mu.Unlock()

	return &proto.SendMessageResponse{
		LogicalTimestamp: s.clock.Get(),
	}, nil
}

func (s *chitChatServer) ReceiveMessages(in *proto.ReceiveMessagesRequest, stream proto.ChitChat_ReceiveMessagesServer) error {
	s.clock.Update(in.LogicalTimestamp)
	logging.Log(logging.Server{}, "receive messages", "receives messages request received", s.clock.Get())

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

	s.clock.Increment()
	logging.Log(logging.Server{}, "receive messages", "receives messages stream ended", s.clock.Get())

	s.mu.Lock()
	delete(s.activeClients, in.Username)
	s.mu.Unlock()

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
	chitChatServer := &chitChatServer{
		activeClients: make(map[string]proto.ChitChat_ReceiveMessagesServer, 0),
		clock:         common.NewLamportClock(),
	}

	grpcServer := grpc.NewServer()
	proto.RegisterChitChatServer(grpcServer, chitChatServer)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for sig := range c {
			msg := fmt.Sprintf("signal: '%s' received, shutting down", sig.String())

			logging.Log(logging.Server{}, "os.Interrupt", msg, chitChatServer.clock.Get())

			grpcServer.GracefulStop()

			logging.Log(logging.Server{}, "os.Interrupt", "server stopped, exiting", chitChatServer.clock.Get())
			os.Exit(0)
		}
	}()

	chitChatServer.start(grpcServer)
}

func (s *chitChatServer) start(server *grpc.Server) {
	network := "tcp"
	port := ":5050"

	listener, err := net.Listen(network, port)
	if err != nil {
		log.Fatalf("failed to create a %s listener on port %s: %v\n", network, port, err)
	}

	if err := server.Serve(listener); err != nil {
		log.Fatalf("failed to start serving requests: %v", err)
	}
}
