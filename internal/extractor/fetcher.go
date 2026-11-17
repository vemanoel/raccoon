package extractor

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
	"raccoon/internal/request"
)

type result struct {
	meta map[string]string
	err  error
}

type RedditOptions struct {
	User string

	After  string
	Before string

	after  time.Time
	before time.Time
}

type postData struct {
	ID          string  `json:"id"`
	CreatedUTC  float64 `json:"created_utc"`
	Author      string  `json:"author"`
	Subreddit   string  `json:"subreddit"`
	URL         string  `json:"url"`
	Domain      string  `json:"domain"`
	SecureMedia *struct {
		RedditVideo *struct {
			HLS_URL *string `json:"hls_url"`
		} `json:"reddit_video"`
		Oembed *struct {
			ThumbnailURL *string `json:"thumbnail_url"`
		} `json:"oembed"`
	} `json:"secure_media"`
	MediaMetadata map[string]struct {
		S struct {
			U string `json:"u"`
		} `json:"s"`
	} `json:"media_metadata"`
}

func (self *RedditOptions) Filter(post *postData) map[string]string {
	if strings.TrimSpace(post.URL) == "" {
		return nil
	}

	meta := make(map[string]string)

    if post.SecureMedia != nil {
		if post.SecureMedia.Oembed.ThumbnailURL == nil {
			return nil
		}

		u, err := url.Parse(*post.SecureMedia.Oembed.ThumbnailURL)
		if err != nil {
			return nil
		}

		if u.Host != "media.redgifs.com" {
			return nil
		}

		if !strings.HasSuffix(u.Path, "-poster.jpg") {
			return nil
		}

		u.Path = strings.TrimSuffix(u.Path, "-poster.jpg")

		meta["name"] = path.Base(u.Path)
		meta["url"] = u.String() + ".mp4"

		return meta
	}

	if post.MediaMetadata != nil {
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

			return meta
		}
	}

	if post.Domain == "i.redd.it" {
		u, err := url.Parse(post.URL)
		if err == nil {
			return nil
		}
		
		meta["name"] = post.ID
		meta["url"] = u.String()

		return meta
	}

	if post.Domain == "v.redd.it" {
	    if post.SecureMedia != nil {
			if post.SecureMedia.RedditVideo.HLS_URL == nil {
				return nil
			}

		    u, err := url.Parse(*post.SecureMedia.RedditVideo.HLS_URL)
		    if err != nil {
		    	return nil
		    }

		    u.Fragment = ""
		    u.RawQuery = ""
		    u.ForceQuery = false

		    meta["site"] = "video-secure_media"

		    meta["name"] = post.ID
		    meta["url"] = u.String()
		    meta["format"] = strings.TrimPrefix(filepath.Ext(u.Path), ".")

		    return meta
		} else {
			u, err := url.Parse(post.URL)
			if err != nil {
				return nil
			}

			meta["name"] = post.ID
			meta["url"] = u.String()
			meta["format"] = strings.TrimPrefix(filepath.Ext(u.Path), ".")

			return meta
		}
	}

	return nil
}

func (self *RedditOptions) fetchRedditApi(ch chan<- result) {
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

	req := request.Request{
		Cookies: cookies,
		Limiter: &request.RequestLimiter{
			MinInterval: time.Second,
			MaxInterval: time.Second * 10,
		},
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
				self.User,
				"submitted",
			),
			RawQuery: q.Encode(),
		}

		resp, err := req.Get(u.String())
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

func (self *RedditOptions) fetchArcticShiftApi(ch chan<- result) {
	var req request.Request

	before := time.Now().UTC().Unix()

	for {
		q := url.Values{
			"limit":  {"auto"},
			"sort":   {"desc"},
			"author": {self.User},
			"before": {strconv.FormatInt(before, 10)},
		}

		u := &url.URL{
			Scheme:   "https",
			Host:     "arctic-shift.photon-reddit.com",
			Path:     "api/posts/search",
			RawQuery: q.Encode(),
		}

		resp, err := req.Get(u.String())
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
	err := self.Validate()
	if err != nil {
		return nil, err
	}

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

	count := 0

	for item := range ch {
		if item.err != nil {
			return nil, item.err
		}

		count++

		u := item.meta["url"]

		if _, ok := seen[u]; ok {
			continue
		}

		seen[u] = nil
		res = append(res, item.meta)
	}

	fmt.Println(count)

	return res, nil
}
