package reddit

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

type Config struct {
	Arctic     bool
	Target   string
	Flairs     string
	Subreddits string
	Since      string
	Until      string
	MediaTypes string
}

type parsedConfig struct {
	arctic     bool
	target	   string
	flairs     []string
	subreddits []string
	mediaTypes []string
	since      *time.Time
	until      *time.Time
}

func (self Config) parseTarget() (string, error) {
	if len(self.Target) > 20 || len(self.Target) < 3 {
		return "", fmt.Errorf(
			"username must be between 3 and 20 chars long",
		)
	}

	for _, c := range self.Target {
		switch {
		case c >= 'A' && c <= 'Z':
		case c >= 'a' && c <= 'z':
		case c >= '0' && c <= '9':
		case c == '-' || c == '_':
		default:
			return "", fmt.Errorf(
				"username contains invalid characters",
			)
		}
	}

	return self.Target, nil
}

func (self Config) parseFlairs() ([]string, error) {
	var res []string

	for _, s := range res {
		if strings.TrimSpace(s) == "" {
			return nil, fmt.Errorf("flair cannot be empty")
		}
		res = append(res, s)
	}

	return res, nil
}

func (self Config) parseSubreddits() ([]string, error) {
	var res []string

	for _, s := range strings.Split(self.Subreddits, ",") {
		s = strings.TrimSpace(s)

		if s == "" {
			continue
		}

		if strings.ContainsAny(s, " \t\n\r") {
			return nil, fmt.Errorf("spaces in subreddit name %q", s)
		}

		res = append(res, s)
	}

	return res, nil
}

func (self Config) parseMediaTypes() ([]string, error) {
	allowed := []string{"gif", "image", "gallery", "video"}

	self.MediaTypes = strings.TrimSpace(self.MediaTypes)

	if self.MediaTypes == "" {
		return allowed, nil
	}

	var res []string

	for _, part := range strings.Split(self.MediaTypes, ",") {
		part = strings.TrimSpace(part)

		if slices.Contains(allowed, part) {
			res = append(res, part)
		} else {
			return nil, fmt.Errorf(
				"invalid media type %q (allowed: %s)",
				part,
				strings.Join(allowed, ", "),
			)
		}
	}

	return res, nil
}

func (self Config) parsePeriod(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}

	t, err := time.ParseInLocation(
		"02/01/2006 03:04:05 PM",
		s,
		time.UTC,
	)
	if err != nil {
		return nil, err
	}

	return &t, nil
}

func (self Config) parseSince() (*time.Time, error) {
	return self.parsePeriod(self.Since)
}

func (self Config) parseUntil() (*time.Time, error) {
	return self.parsePeriod(self.Until)
}

func (self Config) parse() (*parsedConfig, error) {
	target, err := self.parseTarget()
	if err != nil {
		return nil, err
	}

	flairs, err := self.parseFlairs()
	if err != nil {
		return nil, err
	}

	subreddits, err := self.parseSubreddits()
	if err != nil {
		return nil, err
	}

	since, err := self.parseSince()
	if err != nil {
		return nil, err
	}

	until, err := self.parseUntil()
	if err != nil {
		return nil, err
	}

	mediaTypes, err := self.parseMediaTypes()
	if err != nil {
		return nil, err
	}

	return &parsedConfig{
		target:   	target,
		flairs:     flairs,
		subreddits: subreddits,
		since:      since,
		until:      until,
		mediaTypes: mediaTypes,
	}, nil
}
