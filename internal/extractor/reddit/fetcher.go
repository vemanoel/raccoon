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

type result struct {
    meta map[string]string
    err  error
}

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

func (self *RedditOptions) Filter(post *postData) map[string]string {
	meta := map[string]string{
		"site":        "reddit",
		"uploader":    post.Author,
		"upload_date": time.Unix(int64(post.CreatedUTC), 0).UTC().Format("01-02-2006"),
	}

	switch post.Domain {
	case "redgifs.com":
		if post.SecureMedia == nil {
			return nil
		}

		u, err := url.Parse(post.SecureMedia.Oembed.Thumbnail)
		if err != nil {
			return nil
		}

		if u.Host != "media.redgifs.com" {
			return nil
		}

		u.Path = strings.Replace(u.Path, "-poster.jpg", "", 1)

		meta["name"] = path.Base(u.Path)
		meta["url"] = u.String() + ".mp4"
		meta["format"] = "mp4"
	case "reddit.com":
		if post.MediaMetadata == nil {
			return nil
		}

		for k, v := range post.MediaMetadata {
			u, err := url.Parse(v.S.U)
			if err != nil {
				return nil
			}

			if u.Host != "preview.redd.it" {
				return nil
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
			return nil
		}

		meta["name"] = post.ID
		meta["url"] = u.String()
		meta["format"] = strings.TrimPrefix(filepath.Ext(u.Path), ".")
	default:
		return nil
	}

	return meta
}

func (self *RedditOptions) fetchRedditApi(ch chan <- result) {
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
		ch <- result{err: err}
		return
	}

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
			ch <- result{err: err}
			return
		}

		req.AddCookies(cookies...)

		resp, err := client.Do(req)
		if err != nil {
			ch <- result{err: err}
			return
		}

		if resp.StatusCode() != http.StatusOK {
			resp.Body().Close()
			ch <- result{
				err: fmt.Errorf(
				  "unexpected status code: %d",
				  resp.StatusCode(),
			    ),
			}
			return
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
			ch <- result{err: err}
			return
		}

		for i := range json.Data.Children {
			child := json.Data.Children[i]
			meta := self.Filter(&child.Data)
			if meta != nil {
				ch <- result{meta: meta}
			}
		}

		if json.Data.After == nil {
			break
		}

		after = json.Data.After
	}
}

func (self *RedditOptions) fetchArcticShiftApi(ch chan <- result) {
	var client httpclient.Client

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
			ch <- result{err: err}
			return
		}

		resp, err := client.Do(req)
		if err != nil {
			ch <- result{err: err}
			return
		}

		if resp.StatusCode() != http.StatusOK {
			resp.Body().Close()
			ch <- result{
				err: fmt.Errorf(
				   "unexpected status code: %d",
				    resp.StatusCode(),
			    ),
			}
			return
		}

		var json struct {
			Error *string    `json:"error"`
			Data  []postData `json:"data"`
		}

		err = resp.JSON(&json)

		resp.Body().Close()

		if err != nil {
			ch <- result{err: err}
			return
		}

		if json.Error != nil {
			ch <- result{err: errors.New(*json.Error)}
			return
		}

		for i := range json.Data {
			meta := self.Filter(&json.Data[i])
			if meta != nil {
				ch <- result{meta: meta}
			}
		}

		if len(json.Data) == 0 || len(json.Data) < 100 {
			break
		}

		before = int64(json.Data[len(json.Data)-1].CreatedUTC)
	}
}

func (self *RedditOptions) Extract() ([]map[string]string, error) {
	var wg sync.WaitGroup

	wg.Add(2)

	ch := make(chan result, 32)

	go func() {
		defer wg.Done()
		self.fetchRedditApi(ch)
	}()

	go func() {
		defer wg.Done()
		self.fetchArcticShiftApi(ch)
	}()

	go func() {
	    wg.Wait()
		close(ch)
	}()

    var res []map[string]string
    seen := make(map[string]any, len(ch))

    for item := range ch {
		if item.err != nil {
			return nil, item.err
		}

        u := item.meta["url"]

        if _, ok := seen[u]; ok {
             continue
        }

        seen[u] = nil
        res = append(res, item.meta)
    }

	return res, nil
}
