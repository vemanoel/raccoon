package cookies

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"time"
	_ "modernc.org/sqlite"
)

type HostOptions struct {
	Name     string
	Override string
}

type Options struct {
	Names []string
	Host  HostOptions
}

func Get(opts Options) ([]*http.Cookie, error) {
	db, cleanup, err := openDB()
	if err != nil {
		return nil, err
	}
	defer cleanup()

	rows, err := db.Query(`
		SELECT
			name,
			value,
			host,
			path,
			isSecure,
			expiry
		FROM
			moz_cookies
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []*http.Cookie

	for rows.Next() {
		var (
			c http.Cookie

			expiry int64
		)

		err = rows.Scan(
			&c.Name,
			&c.Value,
			&c.Domain,
			&c.Path,
			&c.Secure,
			&expiry,
		)
		if err != nil {
			return nil, err
		}

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

		c.Expires = time.Unix(expiry, 0)

		res = append(res, &c)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return res, nil
}

func openDB() (*sql.DB, func(), error) {
	profile := os.Getenv("RACCOON_BROWSER_PROFILE")
	if profile == "" {
		return nil, nil, fmt.Errorf(
			"RACCOON_BROWSER_PROFILE is not set",
		)
	}

	src := filepath.Join(profile, "cookies.sqlite")

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, nil, err
	}

	tmp := filepath.Join(home, "raccoon", "databases")
	err = os.MkdirAll(tmp, 0755)
	if err != nil {
		return nil, nil, err
	}

	tmp, err = os.MkdirTemp(tmp, "database-*")
	if err != nil {
		return nil, nil, err
	}

	dst := filepath.Join(tmp, filepath.Base(src))

	err = copyFile(src, dst)
	if err != nil {
		os.RemoveAll(tmp)
		return nil, nil, err
	}

	err = copyFile(src+"-wal", dst+"-wal")
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		os.RemoveAll(tmp)
		return nil, nil, err
	}

	err = copyFile(src+"-shm", dst+"-shm")
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		os.RemoveAll(tmp)
		return nil, nil, err
	}

	dsn := fmt.Sprintf("file:%s?mode=ro&immutable=1", dst)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		os.RemoveAll(tmp)
		return nil, nil, err
	}

	cleanup := func() {
		db.Close()
		os.RemoveAll(tmp)
	}

	return db, cleanup, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}

	return out.Close()
}
