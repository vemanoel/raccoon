package request

import (
	"math/rand/v2"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type RequestLimiter struct {
	mu   sync.Mutex
	next time.Time

	MinInterval time.Duration
	MaxInterval time.Duration
}

type Request struct {
	Retries int
	Timeout time.Duration
	Cookies []*http.Cookie
	Headers []*http.Header
	Limiter *RequestLimiter
}

func (self *RequestLimiter) wait() {
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

func (self *Request) Get(url string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	for _, cookie := range self.Cookies {
		req.AddCookie(cookie)
	}

	req.Header = make(http.Header)

	for _, header := range self.Headers {
		for name, vals := range *header {
			for _, val := range vals {
				req.Header.Add(name, val)
			}
		}
	}

	client := http.Client{
		Timeout: self.Timeout,
	}

	for attempt := 1; ; attempt++ {
		self.Limiter.wait()

		resp, err := client.Do(req)
		if err != nil {
			if attempt == self.Retries {
				return nil, err
			}
			continue
		}

		if attempt == self.Retries {
			return resp, nil
		}

		switch resp.StatusCode {
		case 429, 422:
			resp.Body.Close()

			delay := time.Second * 10

			if s := resp.Header.Get("X-Ratelimit-Reset"); s != "" {
				if n, err := strconv.ParseFloat(s, 64); err == nil {
					delay = time.Duration(n * float64(time.Second))
				}
			} else if s := resp.Header.Get("Retry-After"); s != "" {
				if t, err := http.ParseTime(s); err == nil {
					delay = time.Until(t)
				}
				if n, err := strconv.Atoi(s); err == nil {
					delay = time.Duration(n) * time.Second
				}
			}

			time.Sleep(delay)
		case 500, 502, 503, 504:
			resp.Body.Close()
			time.Sleep(time.Duration(min(1<<attempt, 60)) * time.Second)
		default:
			return resp, nil
		}
	}
}
