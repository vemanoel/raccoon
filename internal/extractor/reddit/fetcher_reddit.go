package reddit

import (
	"fmt"
	"net/http"
	"net/url"
	"path"
	"time"

	"raccoon/internal/cookies"
	"raccoon/internal/httpclient"
)

type redditPage struct {
	Data struct {
		After    *string `json:"after"`
		Children []struct {
			Kind string `json:"kind"`
			Data struct {
				ID            string  `json:"id"`
				CreatedUTC    float64 `json:"created_utc"`
				Title         string  `json:"title"`
				URL           string  `json:"url"`
				LinkFlairText string  `json:"link_flair_text"`
				Domain        string  `json:"domain"`
				MediaMetadata map[string]struct {
					S struct {
						U string `json:"u"`
					} `json:"s"`
				} `json:"media_metadata"`
			} `json:"data"`
		} `json:"children"`
	} `json:"data"`
}

func redditRun(user string) (map[string]string, error) {
	client := httpclient.Client{
		Limiter: &httpclient.Limiter{
			MinInterval: time.Second,
			MaxInterval: time.Second * 10,
		},
	}

	cookies, err := cookies.Get(
		cookies.Options{
			Names: []string{
				"reddit_session",
				"token_v2",
			},
			Host: cookies.HostOptions{
				Name:     ".reddit.com",
				Override: "api.reddit.com",
			},
		},
	)
	if err != nil {
		return nil, err
	}

	after, res := new(string), make(map[string]string)

	for after != nil {
		q := url.Values{
			"limit": {"100"},
			"sort":  {"new"},
			"after": {*after},
		}

		u := &url.URL{
			Scheme: "https",
			Host:   "api.reddit.com",
			Path: path.Join(
				"user",
				user,
				"submitted",
			),
			RawQuery: q.Encode(),
		}

		req, err := httpclient.NewRequest(
			http.MethodGet,
			u.String(),
		)
		if err != nil {
			return nil, err
		}

		req.AddCookies(cookies...)

		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode() != 200 {
			return nil, fmt.Errorf(
				"unexpected status code: %d",
				resp.StatusCode(),
			)
		}

		var page redditPage

		err = resp.JSON(&page)

		resp.Close()

		if err != nil {
			return nil, err
		}

		for _, child := range page.Data.Children {
			switch child.Data.Domain {
			case "reddit.com":
				if child.Data.MediaMetadata == nil {
					continue
				}

				for k, v := range child.Data.MediaMetadata {
					u, err := url.Parse(v.S.U)
					if err != nil {
						continue
					}

					u.Host = "i.redd.it"
					u.Fragment = ""
					u.RawQuery = ""
					u.ForceQuery = false

					res[u.String()] = k
				}
			case "v.redd.it", "i.redd.it":
				res[child.Data.URL] = child.Data.ID
			}
		}

		if page.Data.After == nil {
			break
		}

		after = page.Data.After
	}

	return res, nil
}
