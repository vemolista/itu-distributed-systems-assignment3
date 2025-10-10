package common

// See https://martinfowler.com/articles/patterns-of-distributed-systems/lamport-clock.html

type LamportClock struct {
	timestamp int64
}

func (c *LamportClock) Tick(requestTime int64) int64 {
	latestTime := max(c.timestamp, requestTime)
	latestTime += 1
	c.timestamp = latestTime
	return latestTime
}
