package main

import (
	"fmt"

	"raccoon/internal/downloader"
	"raccoon/internal/extractor/reddit"
)

func main() {
	posts, err := reddit.Fetch(reddit.Config{
		Username: "lailaposts",
	})
	if err != nil {
		panic(err)
	}

	for _, media := range posts {
		err := transfer.Download(media.URL, "./downloads")
		fmt.Println(err)
	}
}
