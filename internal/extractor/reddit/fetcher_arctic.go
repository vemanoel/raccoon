package reddit

import (
	"errors"
	"net/url"
	"strconv"
	"time"

	"raccoon/internal/httpclient"
)

type arcticShiftPage struct {
	Error *string `json:"error"`
	Data  []struct {
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
}

func arcticShiftRun(user string) (map[string]string, error) {
	client := httpclient.Client{
		Limiter: &httpclient.Limiter{
			MinInterval: time.Second,
			MaxInterval: time.Second * 5,
		},
	}

	before := time.Now().UTC().Unix()
	res := make(map[string]string)

	for {
		limit := 100

		query := url.Values{
			"limit":  {strconv.Itoa(limit)},
			"sort":   {"desc"},
			"author": {user},
			"before": {strconv.FormatInt(before, 10)},
		}

		u := &url.URL{
			Scheme:   "https",
			Host:     "arctic-shift.photon-reddit.com",
			Path:     "api/posts/search",
			RawQuery: query.Encode(),
		}

		req, err := httpclient.NewRequest("GET", u.String())
		if err != nil {
			return nil, err
		}

		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}

		var page arcticShiftPage

		err = resp.JSON(&page)

		resp.Close()

		if err != nil {
			return nil, err
		}

		if page.Error != nil {
			return nil, errors.New(*page.Error)
		}

		for _, post := range page.Data {
			switch post.Domain {
			case "reddit.com":
				if post.MediaMetadata == nil {
					continue
				}
				for k, v := range post.MediaMetadata {
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
				res[post.URL] = post.ID
			}
		}

		if len(page.Data) == 0 || len(page.Data) < limit {
			break
		}

		before = int64(page.Data[len(page.Data)-1].CreatedUTC)
	}

	return res, nil
}
