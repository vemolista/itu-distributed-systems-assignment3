package main

import (
	"bufio"
	"context"
	"fmt"
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

	resp, err := c.client.Join(context.Background(), &proto.JoinRequest{Username: "john", LogicalTimestamp: c.clock.Get()})
	if err != nil {
		log.Fatalf("failed to join: %v", err)
		return err
	}

	c.clock.Update(resp.LogicalTimestamp)
	return nil
}

func main() {
	conn, err := grpc.NewClient("localhost:5050", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}
	defer conn.Close()
	client := newChitChatClient(conn, "john")

	if err = client.join(); err != nil {
		log.Fatalf("failed to join: %v", err)
	}

	scanner := bufio.NewScanner(os.Stdin)

	for {
		scanner.Scan()
		input := scanner.Text()

		if utf8.RuneCountInString(input) > characterLimit {
			fmt.Println("Message rejected. Maximum of 128 characters allowed.")
			continue
		}

		// client.SendMessage(context.Background(), &proto.SendMessageRequest{
		// 	Message: &proto.ChatMessage{
		// 		Content:  input,
		// 		Username: "john", // TODO
		// 		Type:     proto.MessageType_USER_MESSAGE,
		// 	},
		// 	LogicalTimestamp: 1, // TODO
		// })
	}
}
