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
			FallbackURL string `json:"fallback_url"`
		} `json:"reddit_video"`
		Oembed *struct {
			Thumbnail string `json:"thumbnail_url"`
		} `json:"oembed"`
	} `json:"secure_media"`
	MediaMetadata map[string]struct {
		S struct {
			U string `json:"u"`
		} `json:"s"`
	} `json:"media_metadata"`
}

func (self *RedditOptions) Validate() error {
	const layout = "02/01/2006 15:04:05"

	var err error

	if self.After != "" {
		self.after, err = time.Parse(layout, self.After)
		if err != nil {
			return fmt.Errorf("--after: invalid date format")
		}
	}

	if self.Before != "" {
		self.before, err = time.Parse(layout, self.Before)
		if err != nil {
			return fmt.Errorf("--before: invalid date format")
		}
	} else if !self.after.IsZero() {
		self.before = time.Now().UTC()
	}

	if !self.after.IsZero() && !self.before.IsZero() &&
		self.after.After(self.before) {
		return errors.New("--after cannot be later than --before")
	}

	return nil
}

func (self *RedditOptions) Filter(post *postData) map[string]string {
	created := time.Unix(int64(post.CreatedUTC), 0).UTC()

	if !self.after.IsZero() && created.Before(self.after) {
		return nil
	}

	if !self.before.IsZero() && created.After(self.before) {
		return nil
	}

	meta := map[string]string{
		"site":        "reddit",
		"uploader":    post.Author,
		"upload_date": created.Format("01-02-2006_15-04-05"),
	}

	if post.Domain == "redgifs.com" {
		if post.SecureMedia != nil {
			u, err := url.Parse(post.SecureMedia.Oembed.Thumbnail)
			if err != nil {
				return nil
			}

			// u.Scheme = "https"

			u.Path = strings.Replace(u.Path, "-poster.jpg", "", 1)

			meta["site"] = "redgifs"

			meta["name"] = path.Base(u.Path)
			meta["url"] = u.String() + ".mp4"
			meta["format"] = "mp4"

			return meta
		}
	}

	if post.Domain == "reddit.com" && post.MediaMetadata != nil {
		for k, v := range post.MediaMetadata {
			u, err := url.Parse(v.S.U)
			if err != nil {
				return nil
			}

			u.Host = "i.redd.it"
			u.Fragment = ""
			u.RawQuery = ""
			u.ForceQuery = false

			meta["site"] = "gallery-item"

			meta["name"] = k
			meta["url"] = u.String()
			meta["format"] = strings.TrimPrefix(filepath.Ext(u.Path), ".")

			return meta
		}
	}

	if post.Domain == "i.redd.it" {
		u, err := url.Parse(post.URL)
		if err == nil {
			meta["image"] = "redgifs"

			meta["name"] = post.ID
			meta["url"] = u.String()
			meta["format"] = strings.TrimPrefix(filepath.Ext(u.Path), ".")

			return meta
		}
	}

	if post.Domain == "v.redd.it" {
		if post.SecureMedia != nil {
			u, err := url.Parse(post.SecureMedia.RedditVideo.FallbackURL)
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
