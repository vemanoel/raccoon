package cookies

import (
	"bufio"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
)

func readBrowserCookies() ([]byte, error) {
	const cookiesFile = "cookies.txt"

	data, err := os.ReadFile(cookiesFile)
	if err != nil {
		return nil, err
	}

	err = os.Remove(cookiesFile)
	if err != nil {
		return nil, err
	}

	err = os.Remove("yt-dlp")
	if err != nil {
		return nil, err
	}

	return data, nil
}

func parseBrowserCookies() ([]*http.Cookie, error) {
	data, err := readBrowserCookies()
	if err != nil {
		return nil, err
	}

	var cookies []*http.Cookie

	scanner := bufio.NewScanner(strings.NewReader(string(data)))

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Split(line, "\t")
		if len(fields) != 7 {
			continue
		}

		cookie := &http.Cookie{
			Domain:   fields[0],
			Path:     fields[2],
			Secure:   fields[3] == "TRUE",
			Name:     fields[5],
			Value:    fields[6],
			HttpOnly: false,
		}

		expires, err := strconv.ParseInt(fields[4], 10, 64)
		if err != nil {
			return nil, err
		}

		if expires > 0 {
			cookie.Expires = time.Unix(expires, 0)
		}

		cookies = append(cookies, cookie)
	}

	return cookies, scanner.Err()
}

type HostOptions struct {
	Name     string
	Override string
}

type Options struct {
	Names []string
	Host  HostOptions
}

func Get(opts Options) ([]*http.Cookie, error) {
	var filtered []*http.Cookie

	all, err := parseBrowserCookies()
	if err != nil {
		return nil, err
	}

	for _, c := range all {
		if len(opts.Names) > 0 {
			if !slices.Contains(opts.Names, c.Name) {
				continue
			}
		}

		if len(opts.Host.Name) != 0 {
			if opts.Host.Name != c.Domain {
				continue
			}
		}

		if len(opts.Host.Override) != 0 {
			c.Domain = opts.Host.Override
		}

		filtered = append(filtered, c)
	}

	return filtered, nil
}
