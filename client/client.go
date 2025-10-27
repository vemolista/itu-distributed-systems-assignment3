package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"unicode/utf8"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/vemolista/itu-distributed-systems-assignment3/v2/common"
	"github.com/vemolista/itu-distributed-systems-assignment3/v2/common/logging"
	proto "github.com/vemolista/itu-distributed-systems-assignment3/v2/grpc"
)

const characterLimit = 128

type chitChatClient struct {
	client   proto.ChitChatClient
	clock    common.LamportClock
	logger   *log.Logger
	username string
}

func newChitChatClient(conn *grpc.ClientConn, username string, logger *log.Logger) *chitChatClient {
	return &chitChatClient{
		client:   proto.NewChitChatClient(conn),
		username: username,
		clock:    common.NewLamportClock(),
		logger:   logger,
	}
}

func (c *chitChatClient) join(logger *log.Logger) error {
	c.clock.Increment()

	logging.Log(logging.Client{Username: c.username}, "join", "sending join request", c.clock.Get())
	resp, err := c.client.Join(context.Background(), &proto.JoinRequest{
		Username:         c.username,
		LogicalTimestamp: c.clock.Get(),
	})
	c.clock.Update(resp.LogicalTimestamp)
	logging.Log(logging.Client{Username: c.username}, "join", "join request response received", c.clock.Get())

	if err != nil {
		logger.Fatalf("failed to join: %v", err)
		return err

	}
	c.clock.Update(resp.LogicalTimestamp)

	// Log client join
	c.logger.Printf("[component: client] [client: %s] [event: JOIN] [timestamp: %d] [content: joined chat]",
		c.username, c.clock.Get())

	return nil
}

func (c *chitChatClient) leave(logger *log.Logger) error {
	c.clock.Increment()

	resp, err := c.client.Leave(context.Background(), &proto.LeaveRequest{
		Username:         c.username,
		LogicalTimestamp: c.clock.Get(),
	})
	c.clock.Update(resp.LogicalTimestamp)
	logging.Log(logging.Client{Username: c.username}, "leave", "leave request response received", c.clock.Get())

	if err != nil {
		logger.Fatalf("[client %s]: failed to leave: %v", c.username, err)
		return err
	}

	// Log client leave
	c.logger.Printf("[component: client] [client: %s] [event: LEAVE] [timestamp: %d] [content: left chat]",
		c.username, c.clock.Get())

	return nil
}

func (c *chitChatClient) sendMessage(input string, logger *log.Logger) error {
	c.clock.Increment()

	msg := fmt.Sprintf("sending send message request, content: %s", input)
	logging.Log(logging.Client{Username: c.username}, "send message", msg, c.clock.Get())

	resp, err := c.client.SendMessage(context.Background(), &proto.SendMessageRequest{
		Message: &proto.ChatMessage{
			Username: c.username,
			Content:  input,
			Type:     proto.MessageType_USER_MESSAGE,
		},
		LogicalTimestamp: c.clock.Get(),
	})
	c.clock.Update(resp.LogicalTimestamp)
	logging.Log(logging.Client{Username: c.username}, "send message", "send message request response received", c.clock.Get())

	if err != nil {
		logger.Printf("failed to send message: %v", err)
		return fmt.Errorf("failed to send message: %v", err)
	}
	logger.Printf("[client %s]: message sent: (timestamp: %d) (content: %s)\n", c.username, c.clock.Get(), input)

	return nil
}

func (c *chitChatClient) waitForInput(logger *log.Logger) {
	scanner := bufio.NewScanner(os.Stdin)

	for {
		scanner.Scan()
		input := scanner.Text()

		// Skip empty messages
		if len(input) == 0 {
			continue
		}

		// Check if user wants to leave
		if input == "LEAVE" {
			c.leave(logger)
			fmt.Println("You have left the chat.")
			os.Exit(0)
		}

		if utf8.RuneCountInString(input) > characterLimit {
			logging.Log(logging.Client{Username: c.username}, "user input", "message too long", c.clock.Get())
			fmt.Println("Message rejected. Maximum of 128 characters allowed.")
			continue
		}

		c.sendMessage(input, logger)
	}
}

func formatMessage(msg *proto.ReceiveMessagesResponse) string {
	if msg.Message.Type == proto.MessageType_SYSTEM_JOIN || msg.Message.Type == proto.MessageType_SYSTEM_LEAVE {
		return fmt.Sprintf("[SYSTEM] %d: %s\n", msg.LogicalTimestamp, msg.Message.Content)
	}

	return fmt.Sprintf("[%s] %d: %s\n", msg.Message.Username, msg.LogicalTimestamp, msg.Message.Content)
}

func (c *chitChatClient) receiveMessages(logger *log.Logger) {
	c.clock.Increment()

	logging.Log(logging.Client{Username: c.username}, "receive messages", "sending receive messages request", c.clock.Get())
	stream, err := c.client.ReceiveMessages(
		context.Background(),
		&proto.ReceiveMessagesRequest{
			Username:         c.username,
			LogicalTimestamp: c.clock.Get(),
		},
	)
	logging.Log(logging.Client{Username: c.username}, "receive messages", "receive messages request response received", c.clock.Get())

	if err != nil {
		logger.Fatalf("failed to receive messages: %v", err)
	}
	logger.Printf("[client %s]: receiving messages: (timestamp: %d)\n", c.username, c.clock.Get())

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		c.clock.Update(resp.LogicalTimestamp)
		logging.Log(logging.Client{Username: c.username}, "receive messages", "received response from stream", c.clock.Get())

		if err != nil {
			logger.Printf("failed to read from messages stream: %v", err)
			continue
		}

		fmt.Printf("%s", formatMessage(resp))

		// Log message reception with all required details
		eventType := "USER_MESSAGE"
		if resp.Message.Type == proto.MessageType_SYSTEM_JOIN {
			eventType = "SYSTEM_JOIN"
		} else if resp.Message.Type == proto.MessageType_SYSTEM_LEAVE {
			eventType = "SYSTEM_LEAVE"
		}

		c.logger.Printf("[component: client] [client: %s] [event: %s] [timestamp: %d] [content: %s]",
			c.username, eventType, c.clock.Get(), resp.Message.Content)
	}

	c.leave()
	os.Exit(0)
}

func main() {
	file, err := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err)
	}
	logger := log.New(file, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)

	conn, err := grpc.NewClient("localhost:5050", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Fatalf("failed to create client: %v", err)
	}
	defer conn.Close()

	fmt.Printf("select name: ")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	username := scanner.Text()

	client := newChitChatClient(conn, username, logger)

	// Log client start
	client.logger.Printf("[component: client] [client: %s] [event: START] [timestamp: %d] [content: client started]",
		client.username, client.clock.Get())

	if err = client.join(logger); err != nil {
		logger.Fatalf("failed to join: %v", err)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for sig := range c {
			// Log client shutdown
			client.logger.Printf("[component: client] [client: %s] [event: SHUTDOWN] [timestamp: %d] [signal: %s]",
				client.username, client.clock.Get(), sig.String())
			log.Printf("[client %s] shutting down: %s\n", client.username, sig.String())

			client.leave(logger)

			os.Exit(0)
		}
	}()

	go client.receiveMessages(logger)
	client.waitForInput(logger)
}
