package main

import (
	"flag"
	"fmt"
	"os"

	"raccoon/internal/downloader"
	"raccoon/internal/extractor"
)

var browser string
var profile string
var username string

func init() {
	flag.StringVar(&browser, "browser", os.Getenv("BROWSER"), "")
	flag.StringVar(&profile, "profile", os.Getenv("PROFILE"), "")
	flag.StringVar(&username, "username", os.Getenv("USERNAME"), "")
	flag.Parse()
}

func main() {
	extractor := reddit.RedditOptions{
		Username: username,
		// After:  "12/11/2026 00:00:00",
		// Before: "12/11/2026 22:00:00",
	}
	medias, err := extractor.Extract()
	if err != nil {
		panic(err)
	}

	for _, media := range medias {
		err := downloader.Download(
			"./downloads/{{.site}}/{{.uploader}}/{{.upload_date}}-{{.name}}.{{.format}}",
			media,
		)
		if err != nil {
			fmt.Println(err, media)
		}
	}
}
