package cookies

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"
)

func ytDlpReleaseAssetName() string {
	if runtime.GOOS == "windows" {
		switch runtime.GOARCH {
		case "386":
			return "yt-dlp_x86.exe"
		case "amd64":
			return "yt-dlp.exe"
		case "arm64":
			return "yt-dlp_arm64.exe"
		}
	}

	if runtime.GOOS == "linux" {
		cmd := exec.Command("ldd", "--version")
		out, _ := cmd.CombinedOutput()
		output := strings.ToLower(string(out))
		useMusl := strings.Contains(output, "musl")

		switch {
		case runtime.GOARCH == "amd64" && useMusl:
			return "yt-dlp_musllinux"
		case runtime.GOARCH == "amd64":
			return "yt-dlp_linux"
		case runtime.GOARCH == "arm64" && useMusl:
			return "yt-dlp_musllinux_aarch64"
		case runtime.GOARCH == "arm64":
			return "yt-dlp_linux_aarch64"
		}
	}

	return "yt-dlp_macos"
}

func ytDlpReleaseAssetURL() string {
	u := url.URL{
		Scheme: "https",
		Host:   "github.com",
		Path: path.Join(
			"yt-dlp",
			"yt-dlp",
			"releases",
			"latest",
			"download",
			ytDlpReleaseAssetName(),
		),
	}

	return u.String()
}

func downloadYtDlpReleaseAsset() error {
	resp, err := http.Get(ytDlpReleaseAssetURL())
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf(
			"unexpected status: %s",
			resp.Status,
		)
	}

	f, err := os.OpenFile(
		"yt-dlp",
		os.O_CREATE|os.O_WRONLY|os.O_TRUNC,
		0755,
	)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func readBrowserCookies() ([]byte, error) {
	err := downloadYtDlpReleaseAsset()
	if err != nil {
		return nil, err
	}

	const cookiesFile = "cookies.txt"

	browserName := os.Getenv("RACCOON_BROWSER_NAME")
	browserProfile := os.Getenv("RACCOON_BROWSER_PROFILE")

	cmd := exec.Command(
		"./yt-dlp",
		"--cookies-from-browser",
		fmt.Sprintf("%s:%s", browserName, browserProfile),
		"--cookies", "cookies.txt",
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		info, statErr := os.Stat(cookiesFile)

		if statErr != nil || info.Size() == 0 {
			return nil, fmt.Errorf("%w\n%s", err, out)
		}
	}

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
