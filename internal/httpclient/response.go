package httpclient

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
)

type Response struct {
	inner *http.Response
}

func (self *Response) StatusCode() int {
	return self.inner.StatusCode
}

func (self *Response) Body() io.ReadCloser {
	return self.inner.Body
}

func (self *Response) ContentLength() int64 {
	return self.inner.ContentLength
}

func (self *Response) Bytes() ([]byte, error) {
	return io.ReadAll(self.inner.Body)
}

func (self *Response) JSON(res any) error {
	rv := reflect.ValueOf(res)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return fmt.Errorf("res must be a non-nil pointer")
	}
	return json.NewDecoder(self.inner.Body).Decode(res)
}
