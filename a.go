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

type Options struct {
	Profile string
	Host    string
	Names   []string
}

func Get(opts Options) ([]*http.Cookie, error) {
	db, cleanup, err := openDB(opts.Profile)
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
		FROM moz_cookies
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

		if opts.Host != c.Domain {
			continue
		}

		if len(opts.Names) > 0 {
			if !slices.Contains(opts.Names, c.Name) {
				continue
			}
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

func openDB(profile string) (*sql.DB, func(), error) {
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
