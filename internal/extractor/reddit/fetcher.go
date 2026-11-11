package reddit

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"raccoon/internal/cookies"
	"raccoon/internal/httpclient"
	"raccoon/internal/models"
)

type postData struct {
	ID            string  `json:"id"`
	CreatedUTC    float64 `json:"created_utc"`
	Author        string  `json:"author"`
	Title         string  `json:"title"`
	URL           string  `json:"url"`
	LinkFlairText string  `json:"link_flair_text"`
	Domain        string  `json:"domain"`
	MediaMetadata map[string]struct {
		S struct {
			U string `json:"u"`
		} `json:"s"`
	} `json:"media_metadata"`
}

type arcticShiftPage struct {
	Error *string    `json:"error"`
	Data  []postData `json:"data"`
}

type redditPage struct {
	Data struct {
		After    *string `json:"after"`
		Children []struct {
			Kind string   `json:"kind"`
			Data postData `json:"data"`
		} `json:"children"`
	} `json:"data"`
}

func redditRun(user string) ([]postData, error) {
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

	after, result := new(string), make([]postData, 0)

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

		if resp.StatusCode() != http.StatusOK {
			return nil, fmt.Errorf(
				"unexpected status code: %d",
				resp.StatusCode(),
			)
		}

		var page redditPage

		err = resp.JSON(&page)

		resp.Body().Close()

		if err != nil {
			return nil, err
		}

		for _, child := range page.Data.Children {
			result = append(result, child.Data)
		}

		if page.Data.After == nil {
			break
		}

		after = page.Data.After
	}

	return result, nil
}

func arcticShiftRun(user string) ([]postData, error) {
	client := httpclient.Client{
		Limiter: &httpclient.Limiter{
			MinInterval: time.Second,
			MaxInterval: time.Second * 5,
		},
	}

	var result []postData
	before := time.Now().UTC().Unix()

	for {
		limit := 100

		q := url.Values{
			"limit":  {strconv.Itoa(limit)},
			"sort":   {"desc"},
			"author": {user},
			"before": {strconv.FormatInt(before, 10)},
		}

		u := &url.URL{
			Scheme:   "https",
			Host:     "arctic-shift.photon-reddit.com",
			Path:     "api/posts/search",
			RawQuery: q.Encode(),
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

		resp.Body().Close()

		if err != nil {
			return nil, err
		}

		if page.Error != nil {
			return nil, errors.New(*page.Error)
		}

		result = append(result, page.Data...)

		if len(page.Data) == 0 || len(page.Data) < limit {
			break
		}

		before = int64(page.Data[len(page.Data)-1].CreatedUTC)
	}

	return result, nil
}

func Fetch(config Config) ([]models.Media, error) {
	cfg, err := config.parse()
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	var redditErr, arcticShiftErr error
	var arcticShiftPosts, redditPosts []postData

	wg.Add(2)

	go func() {
		defer wg.Done()
		arcticShiftPosts, arcticShiftErr = arcticShiftRun(cfg.username)
	}()

	go func() {
		defer wg.Done()
		redditPosts, redditErr = redditRun(cfg.username)
	}()

	wg.Wait()

	if redditErr != nil {
		return nil, redditErr
	}

	if arcticShiftErr != nil {
		return nil, arcticShiftErr
	}

	var result []models.Media
	seen := make(map[string]struct{})

	for _, post := range append(arcticShiftPosts, redditPosts...) {
		if _, ok := seen[post.URL]; ok {
			continue
		}
		seen[post.URL] = struct{}{}

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

				result = append(result, models.Media{
					URL: u.String(),
					Meta: map[string]string{
						"name":      k,
						"uploader":  post.Author,
						"extension": filepath.Ext(u.Path),
					},
				})
			}
		case "v.redd.it", "i.redd.it":
			u, err := url.Parse(post.URL)
			if err != nil {
				continue
			}

			result = append(result, models.Media{
				URL: post.URL,
				Meta: map[string]string{
					"name":      post.ID,
					"uploader":  post.Author,
					"extension": filepath.Ext(u.Path),
				},
			})
		}
	}

	return result, nil
}
