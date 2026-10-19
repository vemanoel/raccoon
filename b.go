package request

import (
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type Request struct {
	inner http.Request
}

func (self *Request) AddCookies(vals ...*http.Cookie) {
	for _, val := range vals {
		self.inner.AddCookie(val)
	}
}

func (self *Request) AddHeaders(vals ...*http.Header) {
	for _, val := range vals {
		for key, values := range *val {
			for _, value := range values {
				self.inner.Header.Add(key, value)
			}
		}
	}
}

func (self *Request) Do(method, rawURL string) (*Response, error) {
	self.inner.Method = method

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	self.inner.URL = parsedURL

	client := http.Client{
		Timeout: time.Minute,
	}

	for retry := 1; ; retry++ {
		resp, err := client.Do(&self.inner)
		if err != nil {
			if retry < 10 {
				time.Sleep(time.Duration(min(1<<retry, 60)) * time.Second)
				continue
			}
			return nil, err
		}

		switch resp.StatusCode {
		case 500, 502, 503, 504:
			if retry == 10 {
				return &Response{resp}, nil
			}

			resp.Body.Close()

			time.Sleep(time.Duration(min(1<<retry, 60)) * time.Second)
		case 429:
			if retry == 10 {
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

			time.Sleep(time.Second * 10)
		default:
			return &Response{resp}, nil
		}
	}
}
