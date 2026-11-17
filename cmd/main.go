package main

import (
	"flag"
	"fmt"
	"os"

	"raccoon/internal/downloader"
	"raccoon/internal/extractor"
)

var username string

func init() {
	flag.StringVar(&username, "username", os.Getenv("USERNAME"), "")
	flag.Parse()
}

func main() {
	opts := extractor.RedditOptions{
		User: username,
	}
	medias, err := opts.Extract()
	if err != nil {
		panic(err)
	}

	for _, media := range medias {
		err := downloader.Download(
			"./downloads/{{.uploader}}/{{.name}}",
			media,
		)
		if err != nil {
			fmt.Println(err, media)
		}
	}
}
