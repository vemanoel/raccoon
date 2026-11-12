package downloader

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"raccoon/internal/httpclient"
)

type progressReader struct {
	reader  io.Reader
	total   int64
	current int64
}

func (self *progressReader) Read(b []byte) (int, error) {
	n, err := self.reader.Read(b)
	self.current += int64(n)

	if self.total > 0 {
		percent := float64(self.current) / float64(self.total) * 100
		fmt.Printf("\r%.2f%% (%d / %d bytes)", percent, self.current, self.total)
	} else {
		fmt.Printf("\r%d bytes baixados", self.current)
	}

	return n, err
}

func parseOutTmpl(s string, meta map[string]string) (string, error) {
	tmpl := template.Must(template.New("out").Parse(s))

	var buf strings.Builder

	err := tmpl.Execute(&buf, map[string]string{
		"site":        meta["site"],
		"uploader":    meta["uploader"],
		"name":        meta["name"],
		"format":      meta["format"],
		"upload_date": meta["upload_date"],
	})
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func Download(outTmpl string, meta map[string]string) error {
	out, err := parseOutTmpl(outTmpl, meta)
	if err != nil {
		return err
	}
	fmt.Println(meta)
	return nil

	var client httpclient.Client

	req, err := httpclient.NewRequest(
		http.MethodGet,
		meta["url"],
	)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body().Close()

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf(
			"unexpected status code: %d",
			resp.StatusCode(),
		)
	}

	file, err := os.Create(filepath.Join(
		out,
		meta["name"]+meta["format"],
	))
	if err != nil {
		return err
	}
	defer file.Close()

	progress := &progressReader{
		reader: resp.Body(),
		total:  resp.ContentLength(),
	}

	_, err = io.Copy(file, progress)
	fmt.Println()

	return err
}
