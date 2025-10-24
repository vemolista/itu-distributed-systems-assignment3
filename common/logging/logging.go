package logging

import (
	"fmt"
	"log"
)

type Component interface {
	// Unexported method ensures only types in this package can implement Component
	component()
}

type Server struct {
}

type Client struct {
	username string
}

func (Server) component() {}
func (Client) component() {}

func format(c Component, message string, timestamp int32) string {
	switch v := c.(type) {
	case Server:
		return fmt.Sprintf("[server] (logical timestamp: %d) (message: %s)\n", timestamp, message)
	case Client:
		return fmt.Sprintf("[client '%s'] (logical timestamp: %d) (message: %s)\n", v.username, timestamp, message)
	default:
		return fmt.Sprintf("[unknown] (logical timestamp: %d) (message: %s)\n", timestamp, message)
	}
}

func Log(c Component, message string, timestamp int32) {
	logMsg := format(c, message, timestamp)

	log.Printf("%s", logMsg)
}
