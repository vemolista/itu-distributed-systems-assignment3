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
	username string
}

func newChitChatClient(conn *grpc.ClientConn, username string) *chitChatClient {
	return &chitChatClient{
		client:   proto.NewChitChatClient(conn),
		username: username,
		clock:    common.NewLamportClock(),
	}
}

func (c *chitChatClient) join() error {
	c.clock.Increment()

	logging.Log(logging.Client{Username: c.username}, "join", "sending join request", c.clock.Get())
	resp, err := c.client.Join(context.Background(), &proto.JoinRequest{
		Username:         c.username,
		LogicalTimestamp: c.clock.Get(),
	})
	c.clock.Update(resp.LogicalTimestamp)
	logging.Log(logging.Client{Username: c.username}, "join", "join request response received", c.clock.Get())

	if err != nil {
		msg := fmt.Sprintf("failed to join: %v", err)
		logging.Log(logging.Client{Username: c.username}, "join", msg, c.clock.Get())
		return fmt.Errorf("%s", msg)
	}

	logging.Log(logging.Client{Username: c.username}, "join", "joined", c.clock.Get())
	return nil
}

func (c *chitChatClient) leave() error {
	c.clock.Increment()
	logging.Log(logging.Client{Username: c.username}, "leave", "sending leave request", c.clock.Get())

	resp, err := c.client.Leave(context.Background(), &proto.LeaveRequest{
		Username:         c.username,
		LogicalTimestamp: c.clock.Get(),
	})
	c.clock.Update(resp.LogicalTimestamp)
	logging.Log(logging.Client{Username: c.username}, "leave", "leave request response received", c.clock.Get())

	if err != nil {
		msg := fmt.Sprintf("failed to leave: %v", err)
		logging.Log(logging.Client{Username: c.username}, "leave", msg, c.clock.Get())
		return fmt.Errorf("%s", msg)
	}

	return nil
}

func (c *chitChatClient) sendMessage(input string) error {
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
		msg := fmt.Sprintf("failed to send message: %v", err)
		logging.Log(logging.Client{Username: c.username}, "send message", msg, c.clock.Get())
		return fmt.Errorf("%s", msg)
	}

	return nil
}

func (c *chitChatClient) waitForInput() {
	scanner := bufio.NewScanner(os.Stdin)

	for {
		scanner.Scan()
		input := scanner.Text()

		if utf8.RuneCountInString(input) > characterLimit {
			logging.Log(logging.Client{Username: c.username}, "user input", "message too long", c.clock.Get())
			fmt.Println("Message rejected. Maximum of 128 characters allowed.")
			continue
		}

		c.sendMessage(input)
	}
}

func formatMessage(msg *proto.ReceiveMessagesResponse) string {
	if msg.Message.Type == proto.MessageType_SYSTEM_JOIN || msg.Message.Type == proto.MessageType_SYSTEM_LEAVE {
		return fmt.Sprintf("[SYSTEM] %d: %s\n", msg.LogicalTimestamp, msg.Message.Content)
	}

	return fmt.Sprintf("[%s] %d: %s\n", msg.Message.Username, msg.LogicalTimestamp, msg.Message.Content)
}

func (c *chitChatClient) receiveMessages() {
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
		msg := fmt.Sprintf("failed to receive messages: %v", err)
		logging.Log(logging.Client{Username: c.username}, "receive messages", msg, c.clock.Get())
	}

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		c.clock.Update(resp.LogicalTimestamp)
		logging.Log(logging.Client{Username: c.username}, "receive messages", "received response from stream", c.clock.Get())

		if err != nil {
			msg := fmt.Sprintf("failed to read message from stream: %v", err)
			logging.Log(logging.Client{Username: c.username}, "receive messages", msg, c.clock.Get())
			continue
		}

		fmt.Printf("%s", formatMessage(resp))
	}
}

func (c *chitChatClient) gracefulStop(sig os.Signal) {
	if sig == nil {
		log.Printf("[client %s] shutting down\n", c.username)
	} else {
		log.Printf("[client %s] shutting down: %s\n", c.username, sig.String())
	}

	c.leave()
	os.Exit(0)
}

func main() {

	conn, err := grpc.NewClient("localhost:5050", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}
	defer conn.Close()

	fmt.Printf("select name: ")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	username := scanner.Text()

	client := newChitChatClient(conn, username)
	logging.Log(logging.Client{Username: client.username}, "client startup", "new client started", client.clock.Get())

	if err = client.join(); err != nil {
		client.gracefulStop(nil)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for sig := range c {
			client.gracefulStop(sig)
		}
	}()

	go client.receiveMessages()
	client.waitForInput()
}
