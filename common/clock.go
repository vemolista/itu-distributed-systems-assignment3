package common

// See https://martinfowler.com/articles/patterns-of-distributed-systems/lamport-clock.html

type LamportClock struct {
	timestamp int64
}

func (c *LamportClock) Increment() int64 {
	c.timestamp += 1
	return c.timestamp
}

func (c *LamportClock) Update(requestTime int64) int64 {
	c.timestamp = max(c.timestamp, requestTime) + 1
	return c.timestamp
}

func (c *LamportClock) Get() int64 {
	return c.timestamp
}
