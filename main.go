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

	for _, media := range medias {
		err := downloader.Download(
			"./downloads/{{.site}}/{{.uploader}}/{{.upload_date}}-{{.name}}.{{.format}}",
			media,
		)
		if err != nil {
			fmt.Println(err)
		}
	}

	fmt.Println(len(medias))
}
