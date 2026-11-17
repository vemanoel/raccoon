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
	extractor := reddit.RedditOptions{
		Username: username,
	}
	medias, err := extractor.Extract()
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
