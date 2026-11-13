package main

import (
	"fmt"

	"raccoon/internal/downloader"
	"raccoon/internal/extractor/reddit"
)

func main() {
	extractor := reddit.RedditOptions{
		Username: "lailaposts",
	}
	medias, err := extractor.Extract()
	if err != nil {
		panic(err)
	}

	count := 0
	for _, media := range medias {
		err := downloader.Download(
			"./downloads/{{.site}}/{{.uploader}}/{{.upload_date}}-{{.name}}.{{.format}}",
			media,
		)
		if err != nil {
			continue
		}
		count++
	}

	fmt.Println(count)
}
