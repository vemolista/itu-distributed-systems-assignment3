package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"unicode/utf8"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/vemolista/itu-distributed-systems-assignment3/v2/common"
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

	resp, err := c.client.Join(context.Background(), &proto.JoinRequest{Username: c.username, LogicalTimestamp: c.clock.Get()})
	if err != nil {
		log.Fatalf("failed to join: %v", err)
		return err
	}

	c.clock.Update(resp.LogicalTimestamp)
	return nil
}

func (c *chitChatClient) sendMessage(input string) error {
	c.clock.Increment()

	resp, err := c.client.SendMessage(context.Background(), &proto.SendMessageRequest{
		Message: &proto.ChatMessage{
			Username: c.username,
			Content:  input,
			Type:     proto.MessageType_USER_MESSAGE,
		},
		LogicalTimestamp: c.clock.Get(),
	})

	if err != nil {
		log.Printf("failed to send message: %v", err)
		return fmt.Errorf("failed to send message: %v", err)
	}

	c.clock.Update(resp.LogicalTimestamp)
	return nil
}

func (c *chitChatClient) waitForInput() {
	scanner := bufio.NewScanner(os.Stdin)

	for {
		scanner.Scan()
		input := scanner.Text()

		if utf8.RuneCountInString(input) > characterLimit {
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
	stream, err := c.client.ReceiveMessages(context.Background(), &proto.ReceiveMessagesRequest{Username: c.username, LogicalTimestamp: c.clock.Get()})
	if err != nil {
		log.Fatalf("failed to receive messages: %v", err)
	}

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}

		if err != nil {
			log.Printf("failed to read from messages stream: %v", err)
			continue
		}

		fmt.Printf("%s", formatMessage(resp))
	}
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

	if err = client.join(); err != nil {
		log.Fatalf("failed to join: %v", err)
	}

	go client.receiveMessages()
	client.waitForInput()
}
