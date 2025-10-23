package common

import (
	"fmt"
	"sync"
)

// See https://martinfowler.com/articles/patterns-of-distributed-systems/lamport-clock.html

type LamportClock struct {
	timestamp int64
	mu        sync.Mutex
}

func NewLamportClock() LamportClock {
	return LamportClock{}
}

func (c *LamportClock) Increment() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.timestamp += 1

	fmt.Printf("clock incremented %d\n", c.timestamp)
	return c.timestamp
}

func (c *LamportClock) Update(requestTime int64) int64 {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.timestamp = max(c.timestamp, requestTime) + 1

	fmt.Printf("clock updated %d\n", c.timestamp)
	return c.timestamp
}

func (c *LamportClock) Get() int64 {
	return c.timestamp
}
