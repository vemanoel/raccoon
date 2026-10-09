package extractor

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"raccoon/internal/cookies"
	"raccoon/internal/httpclient"
)

type RedditExtractor struct {
	Browser string
	Output  string
	User    string
	Period  string
	Flairs  string
	Medias  string

	client httpclient.Client
}

type redditPage struct {
	Data struct {
		After    *string `json:"after"`
		Children []struct {
			Kind string `json:"kind"`
			Data struct {
				CreatedUTC    float64 `json:"created_utc"`
				Title         string  `json:"title"`
				Selftext      string  `json:"selftext"`
				LinkFlairText string  `json:"link_flair_text"`
				Domain        string  `json:"domain"`
				URL           string  `json:"url"`
				SecureMedia   *struct {
					Oembed struct {
						ThumbnailURL string `json:"thumbnail_url"`
					} `json:"oembed"`
				} `json:"secure_media"`
				MediaMetadata *map[string]struct {
					S struct {
						U string `json:"u"`
					} `json:"s"`
				} `json:"media_metadata"`
			} `json:"data"`
		} `json:"children"`
	} `json:"data"`
}

func (self *RedditExtractor) authenticate() error {
	cookies, err := cookies.Get(cookies.Options{
		Browser: self.Browser,
		Domain:  "reddit.com",
	})
	if err != nil {
		return err
	}

	self.client.Cookies = nil

	for _, c := range cookies {
		switch c.Name {
		case "reddit_session", "token_v2":
		default:
			continue
		}

		c.Domain = "api.reddit.com"

		self.client.Cookies = append(self.client.Cookies, c)
	}

	resp, err := self.client.Req("GET", "https://api.reddit.com/api/me.json")
	if err != nil {
		return err
	}

	if !resp.IsOK() {
		return fmt.Errorf("authentication failed: %v", resp.StatusCode())
	}

	return nil
}

func (self *RedditExtractor) baseEndpoint() (*url.URL, error) {
	re := regexp.MustCompile(`^[ru]/[^/]+$`)
	if !re.MatchString(self.User) {
		return nil, errors.New("invalid target")
	}

	u := url.URL{
		Scheme: "https",
		Host:   "api.reddit.com",
		Path:   self.User,
	}

	if strings.HasPrefix(self.User, "r/") {
		u.Path += "/new"
		u.RawQuery = "limit=100"
	} else {
		u.Path += "/submitted"
		u.RawQuery = "limit=100&sort=new"
	}

	return &u, nil
}

func (self *RedditExtractor) Run() ([]string, error) {
	err := self.authenticate()
	if err != nil {
		return nil, err
	}

	ep, err := self.baseEndpoint()
	if err != nil {
		return nil, err
	}

	urls := make([]string, 0)

	for ep != nil {
		resp, err := self.client.Req("GET", ep.String())
		if err != nil {
			return nil, err
		}

		page, err := resp.JSON[redditPage]()
		if err != nil {
			return nil, err
		}

		for _, child := range page.Data.Children {
			// fmt.Println(child.Data.Title)
			// created := time.Unix(int64(child.Data.CreatedUTC), 0)
			// if !self.Until.IsZero() && created.After(self.Until) {
			// 	continue
			// }
			// if !self.Since.IsZero() && created.Before(self.Since) {
			// 	break
			// }
			urls = append(urls, child.Data.URL)
		}

		if page.Data.After == nil {
			break
		}

		q := ep.Query()
		q.Set("after", *page.Data.After)
		ep.RawQuery = q.Encode()
	}

	return urls, nil
}
