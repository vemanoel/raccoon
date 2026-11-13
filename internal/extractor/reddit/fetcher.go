package reddit

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"raccoon/internal/cookies"
	"raccoon/internal/httpclient"
)

type RedditOptions struct {
	Username string

	Since time.Time
	Until time.Time
}

type postData struct {
	ID          string  `json:"id"`
	CreatedUTC  float64 `json:"created_utc"`
	Author      string  `json:"author"`
	Subreddit   string  `json:"subreddit"`
	URL         string  `json:"url"`
	Domain      string  `json:"domain"`
	SecureMedia *struct {
		Oembed struct {
			Thumbnail string `json:"thumbnail_url"`
		} `json:"oembed"`
	} `json:"secure_media"`
	MediaMetadata map[string]struct {
		S struct {
			U string `json:"u"`
		} `json:"s"`
	} `json:"media_metadata"`
}

func (self *RedditOptions) Filter(post postData) (map[string]string, bool) {
	meta := map[string]string{
		"site":        "reddit",
		"uploader":    post.Author,
		"upload_date": time.Unix(int64(post.CreatedUTC), 0).UTC().Format("01-02-2006"),
	}

	switch post.Domain {
	case "redgifs.com":
		if post.SecureMedia == nil {
			return nil, false
		}

		u, err := url.Parse(post.SecureMedia.Oembed.Thumbnail)
		if err != nil {
			return nil, false
		}

		if u.Host != "media.redgifs.com" {
			return nil, false
		}

		u.Path = strings.Replace(u.Path, "-poster.jpg", "", 1)

		meta["name"] = path.Base(u.Path)
		meta["url"] = u.String() + ".mp4"
		meta["format"] = "mp4"
	case "reddit.com":
		if post.MediaMetadata == nil {
			return nil, false
		}

		for k, v := range post.MediaMetadata {
			u, err := url.Parse(v.S.U)
			if err != nil {
				return nil, false
			}

			if u.Host != "preview.redd.it" {
				return nil, false
			}

			u.Host = "i.redd.it"
			u.Fragment = ""
			u.RawQuery = ""
			u.ForceQuery = false

			meta["name"] = k
			meta["url"] = u.String()
			meta["format"] = strings.TrimPrefix(filepath.Ext(u.Path), ".")
		}
	case "v.redd.it", "i.redd.it":
		u, err := url.Parse(post.URL)
		if err != nil {
			return nil, false
		}

		meta["name"] = post.ID
		meta["url"] = u.String()
		meta["format"] = strings.TrimPrefix(filepath.Ext(u.Path), ".")
	default:
		return nil, false
	}

	return meta, true
}

func (self *RedditOptions) fetchRedditApi() ([]map[string]string, error) {
	client := httpclient.Client{
		Limiter: &httpclient.Limiter{
			MinInterval: time.Second,
			MaxInterval: time.Second * 10,
		},
	}

	cookies, err := cookies.Get(
		cookies.Options{
			Names: []string{
				"token_v2",
				"reddit_session",
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

	var res []map[string]string
	after := new(string)

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
				self.Username,
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
			resp.Body().Close()
			return nil, fmt.Errorf(
				"unexpected status code: %d",
				resp.StatusCode(),
			)
		}

		var json struct {
			Data struct {
				After    *string `json:"after"`
				Children []struct {
					Kind string   `json:"kind"`
					Data postData `json:"data"`
				} `json:"children"`
			} `json:"data"`
		}

		err = resp.JSON(&json)

		resp.Body().Close()

		if err != nil {
			return nil, err
		}

		for _, child := range json.Data.Children {
			if meta, ok := self.Filter(child.Data); ok {
				res = append(res, meta)
			}
		}

		if json.Data.After == nil {
			break
		}

		after = json.Data.After
	}

	return res, nil
}

func (self *RedditOptions) fetchArcticShiftApi() ([]map[string]string, error) {
	var client httpclient.Client

	var res []map[string]string
	before := time.Now().UTC().Unix()

	for {
		q := url.Values{
			"limit":  {"auto"},
			"sort":   {"desc"},
			"author": {self.Username},
			"before": {strconv.FormatInt(before, 10)},
		}

		u := &url.URL{
			Scheme:   "https",
			Host:     "arctic-shift.photon-reddit.com",
			Path:     "api/posts/search",
			RawQuery: q.Encode(),
		}

		req, err := httpclient.NewRequest(
			http.MethodGet,
			u.String(),
		)
		if err != nil {
			return nil, err
		}

		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode() != http.StatusOK {
			resp.Body().Close()
			return nil, fmt.Errorf(
				"unexpected status code: %d",
				resp.StatusCode(),
			)
		}

		var json struct {
			Error *string    `json:"error"`
			Data  []postData `json:"data"`
		}

		err = resp.JSON(&json)

		resp.Body().Close()

		if err != nil {
			return nil, err
		}

		if json.Error != nil {
			return nil, errors.New(*json.Error)
		}

		for _, post := range json.Data {
			if meta, ok := self.Filter(post); ok {
				res = append(res, meta)
			}
		}

		if len(json.Data) == 0 || len(json.Data) < 100 {
			break
		}

		before = int64(json.Data[len(json.Data)-1].CreatedUTC)
	}

	return res, nil
}

func (self *RedditOptions) Extract() ([]map[string]string, error) {
	var wg sync.WaitGroup
	var a, b []map[string]string
	var aErr, bErr error

	wg.Add(2)

	go func() {
		defer wg.Done()
		a, aErr = self.fetchRedditApi()
	}()

	go func() {
		defer wg.Done()
		b, bErr = self.fetchArcticShiftApi()
	}()

	wg.Wait()

	if aErr != nil {
		return nil, aErr
	}

	if bErr != nil {
		return nil, bErr
	}

	var res []map[string]string
	seen := make(map[string]struct{})

	for _, meta := range append(a, b...) {
		if _, ok := seen[meta["url"]]; ok {
			continue
		}

		seen[meta["url"]] = struct{}{}

		res = append(res, meta)
	}

	return res, nil
}
