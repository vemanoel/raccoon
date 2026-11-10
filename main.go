package main

import (
	"fmt"

	"raccoon/internal/extractor/reddit"
)

func main() {
	// var validURLs []string

	// client := http.Client{
	// 	Timeout: 30 * time.Second,
	// }

	// for _, rawURL := range urls {
	// 	resp, err := client.Get(rawURL.URL)
	// 	if err != nil {
	// 		continue
	// 	}

	// 	resp.Body.Close()

	// 	if resp.StatusCode == http.StatusOK {
	// 		fmt.Println(rawURL.URL)
	// 		validURLs = append(validURLs, rawURL.URL)
	// 	}
	// }

	// fmt.Println(len(validURLs))

	// return

	posts, err := reddit.Run(reddit.Config{
		Username: "lailaposts",
	})
	if err != nil {
		panic(err)
	}

	for url, name := range posts {
		fmt.Println(url, name)
	}
}
