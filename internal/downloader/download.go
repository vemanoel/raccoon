package transfer

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"raccoon/internal/httpclient"
)

var client = httpclient.Client{
	Limiter: &httpclient.Limiter{
		MinInterval: time.Second,
		MaxInterval: time.Second * 5,
	},
}

type ProgressReader struct {
	Reader  io.Reader
	Total   int64
	Current int64
}

func (p *ProgressReader) Read(b []byte) (int, error) {
	n, err := p.Reader.Read(b)
	p.Current += int64(n)

	if p.Total > 0 {
		percent := float64(p.Current) / float64(p.Total) * 100
		fmt.Printf("\r%.2f%% (%d / %d bytes)", percent, p.Current, p.Total)
	} else {
		fmt.Printf("\r%d bytes baixados", p.Current)
	}

	return n, err
}

func Download(url, path string) error {
	req, err := httpclient.NewRequest("GET", url)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body().Close()

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode())
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	progress := &ProgressReader{
		Reader: resp.Body(),
		Total:  resp.ContentLength(),
	}

	_, err = io.Copy(file, progress)
	fmt.Println()

	return err
}
