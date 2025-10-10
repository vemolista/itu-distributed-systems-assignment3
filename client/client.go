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

	proto "github.com/vemolista/itu-distributed-systems-assignment3/v2/grpc"
)

const characterLimit = 128

func main() {
	conn, err := grpc.NewClient("localhost:5050", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}

	client := proto.NewChitChatClient(conn)

	resp, err := client.Join(context.Background(), &proto.JoinRequest{Username: "john"})
	if err != nil {
		log.Fatalf("failed to join: %v", err)
	}

	fmt.Printf("%s\n", resp.Message)

	scanner := bufio.NewScanner(os.Stdin)

	for {
		scanner.Scan()
		input := scanner.Text()

		if utf8.RuneCountInString(input) > characterLimit {
			fmt.Println("Message rejected. Maximum of 128 characters allowed.")
			continue
		}

		client.SendMessage(context.Background(), &proto.SendMessageRequest{
			Message: &proto.ChatMessage{
				Content:  input,
				Username: "john", // TODO
				Type:     proto.MessageType_USER_MESSAGE,
			},
			LogicalTimestamp: 1, // TODO
		})
	}
}
