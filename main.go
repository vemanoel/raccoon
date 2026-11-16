package main

// https://www.reddit.com/subreddits/mine

import (
	"flag"
	"fmt"

	"raccoon/internal/downloader"
	"raccoon/internal/extractor/reddit"
)

var user string

func init() {
	flag.StringVar(&user, "user", "", "")
	flag.Parse()
}

func main() {
	extractor := reddit.RedditOptions{
		User: user,
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
