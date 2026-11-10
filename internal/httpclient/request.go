package httpclient

import "net/http"

type Request struct {
	inner *http.Request
}

func NewRequest(method, url string) (*Request, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header = make(http.Header)

	return &Request{req}, nil
}

func (self *Request) AddCookies(cookies ...*http.Cookie) *Request {
	for _, cookie := range cookies {
		self.inner.AddCookie(cookie)
	}

	return self
}

func (self *Request) AddHeaders(headers ...*http.Header) *Request {
	for _, header := range headers {
		for name, vals := range *header {
			for _, val := range vals {
				self.inner.Header.Add(name, val)
			}
		}
	}

	return self
}

func (self *Request) SetUserAgent(ua string) *Request {
	self.inner.Header.Set("User-Agent", ua)
	return self
}

func (self *Request) SetAuthorization(token string) *Request {
	self.inner.Header.Set("Authorization", "Bearer "+token)
	return self
}
