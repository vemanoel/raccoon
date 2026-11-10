package httpclient

import (
	"math/rand/v2"
	"sync"
	"time"
)

type Limiter struct {
	mu   sync.Mutex
	next time.Time

	MinInterval time.Duration
	MaxInterval time.Duration
}

func (self *Limiter) Wait() {
	if self == nil || self.MinInterval <= 0 {
		return
	}

	interval := self.MinInterval
	if self.MaxInterval > self.MinInterval {
		interval += time.Duration(rand.Int64N(
			int64(self.MaxInterval - self.MinInterval),
		))
	}

	self.mu.Lock()

	now := time.Now()
	wait := time.Duration(0)

	if self.next.After(now) {
		wait = time.Until(self.next)
		self.next = self.next.Add(interval)
	} else {
		self.next = now.Add(interval)
	}

	self.mu.Unlock()

	if wait > 0 {
		time.Sleep(wait)
	}
}
