package downloader

import (
	// "fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"raccoon/internal/httpclient"
	"raccoon/internal/progress"
)

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
		return nil
	}

	err = os.MkdirAll(filepath.Dir(out), 0755)
	if err != nil {
		return err
	}

	f, err := os.Create(out)
	if err != nil {
		return err
	}
	defer f.Close()

	bar := progress.New(
		resp.Body(),
		resp.ContentLength(),
	)

	bar.SetDescription(meta["name"] + " " + meta["url"])

	_, err = io.Copy(f, bar)

	if err != nil {
		bar.Finish("failed!")
		return err
	}

	bar.Finish("done!")

	return nil
}
