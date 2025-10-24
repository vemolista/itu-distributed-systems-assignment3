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
	Username string
}

func (Server) component() {}
func (Client) component() {}

func format(c Component, event string, message string, timestamp int64) string {
	switch v := c.(type) {
	case Server:
		return fmt.Sprintf("[server] (event: %s) (logical timestamp: %d) (message: %s)\n", event, timestamp, message)
	case Client:
		return fmt.Sprintf("[client '%s'] (event: %s) (logical timestamp: %d) (message: %s)\n", v.Username, event, timestamp, message)
	default:
		return fmt.Sprintf("[unknown] (event: %s) (logical timestamp: %d) (message: %s)\n", event, timestamp, message)
	}
}

func Log(c Component, event, message string, timestamp int64) {
	log.Printf("%s", format(c, event, message, timestamp))
}
