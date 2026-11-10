package httpclient

import (
	"net/http"
	"strconv"
	"time"
)

type Client struct {
	Limiter *Limiter
	Timeout time.Duration
}

func (self *Client) Do(req *Request) (*Response, error) {
	inner := http.Client{
		Timeout: self.Timeout,
	}

	const maxRetries = 10

	for retry := 1; ; retry++ {
		self.Limiter.Wait()

		resp, err := inner.Do(req.inner)
		if err != nil {
			if retry < maxRetries {
				time.Sleep(time.Second)
				continue
			}
			return nil, err
		}

		switch resp.StatusCode {
		case 500, 502, 503, 504:
			if retry == maxRetries {
				return &Response{resp}, nil
			}

			resp.Body.Close()

			time.Sleep(time.Duration(min(1<<retry, 60)) * time.Second)
		case 429, 422:
			if retry == maxRetries {
				return &Response{resp}, nil
			}

			resp.Body.Close()

			if ra := resp.Header.Get("Retry-After"); ra != "" {
				if t, err := http.ParseTime(ra); err == nil {
					time.Sleep(time.Until(t))
					continue
				}

				if secs, err := strconv.Atoi(ra); err == nil {
					time.Sleep(time.Duration(secs) * time.Second)
					continue
				}
			}

			if xrr := resp.Header.Get("X-Ratelimit-Reset"); xrr != "" {
				if secs, err := strconv.ParseFloat(xrr, 64); err == nil {
					time.Sleep(time.Duration(secs * float64(time.Second)))
					continue
				}
			}

			time.Sleep(time.Second * 10)
		default:
			return &Response{resp}, nil
		}
	}
}
