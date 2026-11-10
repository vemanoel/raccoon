package reddit

import (
	"maps"
	"sync"
)

func Run(config Config) (map[string]string, error) {
	cfg, err := config.parse()
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	var redditErr, arcticShiftErr error
	var posts, arcticShiftPosts, redditPosts map[string]string

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

	maps.Copy(posts, redditPosts)
	maps.Copy(posts, arcticShiftPosts)

	return posts, nil
}
